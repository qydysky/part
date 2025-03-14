package part

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	flate "compress/flate"
	gzip "compress/gzip"

	br "github.com/qydysky/brotli"
	pe "github.com/qydysky/part/errors"
	pio "github.com/qydysky/part/io"
	s "github.com/qydysky/part/strings"
	// "encoding/binary"
)

type Rval struct {
	Method  string
	Url     string
	PostStr string
	Proxy   string
	Retry   int
	// Millisecond
	Timeout int
	// Millisecond
	SleepTime int
	// Millisecond
	WriteLoopTO      int
	JustResponseCode bool
	NoResponse       bool
	// 当Async为true时，Respon、Response必须在Wait()之后读取，否则有DATA RACE可能
	Async   bool
	Cookies []*http.Cookie
	Ctx     context.Context

	SaveToPath string
	// 为避免write阻塞导致panic，请使用此项目io包中的NewPipe()，或在ctx done时，自行关闭pipe writer reader
	SaveToPipe *pio.IOpipe

	RawPipe *RawReqRes

	Header map[string]string
}

const (
	defaultUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"
	defaultAccept = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`
	free          = iota
	running
)

var (
	ErrEmptyUrl         = pe.Action("ErrEmptyUrl")
	ErrCantRetry        = pe.Action("ErrCantRetry")
	ErrNewRequest       = pe.Action("ErrNewRequest")
	ErrClientDo         = pe.Action("ErrClientDo")
	ErrResponFileCreate = pe.Action("ErrResponFileCreate")
	ErrWriteRes         = pe.Action("ErrWriteRes")
	ErrReadRes          = pe.Action("ErrReadRes")
	ErrPostStrOrRawPipe = pe.Action("ErrPostStrOrRawPipe")
	ErrNoDate           = pe.Action("ErrNoDate")
)

type Req struct {
	// 当Async为true时，必须在Wait()之后读取，否则有DATA RACE可能
	Respon []byte
	// 当Async为true时，必须在Wait()之后读取，否则有DATA RACE可能
	Response *http.Response
	UsedTime time.Duration

	cancelP atomic.Pointer[context.CancelFunc]
	state   atomic.Int32

	responFile *os.File
	responBuf  *bytes.Buffer
	err        error
	callTree   string

	copyResBuf []byte
	l          sync.RWMutex
}

func New() *Req {
	return new(Req)
}

func (t *Req) Reqf(val Rval) error {
	t.l.Lock()
	t.state.Store(running)

	pctx, cancelF := t.prepare(&val)
	t.cancelP.Store(&cancelF)

	if !val.Async {
		// 同步
		t.reqfM(pctx, cancelF, val)
		return t.err
	} else {
		//异步
		go t.reqfM(pctx, cancelF, val)
	}

	return nil
}

func (t *Req) reqfM(ctxMain context.Context, cancelMain context.CancelFunc, val Rval) {
	beginTime := time.Now()

	for i := 0; i <= val.Retry; i++ {
		ctx, cancel := t.prepareRes(ctxMain, &val)
		t.err = t.reqf(ctx, val)
		cancel()
		if t.err == nil || IsCancel(t.err) {
			break
		}
		if val.SleepTime != 0 {
			time.Sleep(time.Duration(val.SleepTime * int(time.Millisecond)))
		}
	}

	cancelMain()
	t.updateUseDur(beginTime)
	t.clean(&val)
	t.state.Store(free)
	t.l.Unlock()
}

func (t *Req) reqf(ctx context.Context, val Rval) (err error) {
	var (
		Header map[string]string = val.Header
		client http.Client
	)

	if Header == nil {
		Header = make(map[string]string)
	}

	if val.Proxy != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(val.Proxy)
		}
		client.Transport = &http.Transport{
			Proxy:           proxy,
			IdleConnTimeout: time.Minute,
		}
	} else {
		client.Transport = &http.Transport{
			IdleConnTimeout: time.Minute,
		}
	}

	if val.Url == "" {
		return ErrEmptyUrl.New()
	}

	var body io.Reader
	if len(val.PostStr) > 0 && val.RawPipe != nil {
		return ErrPostStrOrRawPipe.New()
	}
	if val.Retry != 0 && val.RawPipe != nil {
		return ErrCantRetry.New()
	}
	if val.RawPipe != nil {
		body = val.RawPipe
	}

	if len(val.PostStr) > 0 {
		body = strings.NewReader(val.PostStr)
		if _, ok := Header["Content-Type"]; !ok {
			Header["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	req, e := http.NewRequestWithContext(ctx, val.Method, val.Url, body)
	if e != nil {
		return pe.Join(ErrNewRequest.New(), e)
	}

	for _, v := range val.Cookies {
		req.AddCookie(v)
	}

	if _, ok := Header["Accept"]; !ok {
		Header["Accept"] = defaultAccept
	}
	if _, ok := Header["Connection"]; !ok {
		Header["Connection"] = "keep-alive"
	}
	if _, ok := Header["Accept-Encoding"]; !ok {
		Header["Accept-Encoding"] = "gzip, deflate, br"
	}
	if val.SaveToPath != "" || val.SaveToPipe != nil {
		Header["Accept-Encoding"] = "identity"
	}
	if _, ok := Header["User-Agent"]; !ok {
		Header["User-Agent"] = defaultUA
	}

	for k, v := range Header {
		req.Header.Set(k, v)
	}

	resp, e := client.Do(req)

	if e != nil {
		return pe.Join(ErrClientDo.New(), e)
	}

	if v, ok := Header["Connection"]; ok && strings.ToLower(v) != "keep-alive" {
		defer client.CloseIdleConnections()
	}

	t.Response = resp

	if val.JustResponseCode {
		return
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var ws []io.Writer
	if val.SaveToPath != "" {
		t.responFile, e = os.Create(val.SaveToPath)
		if e != nil {
			t.responFile.Close()
			return pe.Join(err, ErrResponFileCreate.New(), e)
		}
		ws = append(ws, t.responFile)
	}
	if val.SaveToPipe != nil {
		ws = append(ws, val.SaveToPipe)
	}
	if val.RawPipe != nil {
		ws = append(ws, val.RawPipe)
	}
	if !val.NoResponse {
		ws = append(ws, t.responBuf)
	}

	var resReadCloser = resp.Body
	if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 {
		switch compress_type[0] {
		case `br`:
			resReadCloser = pio.RWC{R: br.NewReader(resp.Body).Read}
		case `gzip`:
			resReadCloser, _ = gzip.NewReader(resp.Body)
		case `deflate`:
			resReadCloser = flate.NewReader(resp.Body)
		default:
		}
	}

	writeLoopTO := val.WriteLoopTO
	if writeLoopTO == 0 {
		if val.Timeout > 0 {
			writeLoopTO = val.Timeout + 500
		} else {
			writeLoopTO = 1000
		}
	}

	// io copy
	errChan := make(chan error, 3)
	errChan <- pio.WithCtxCopy(
		req.Context(),
		t.callTree,
		t.copyResBuf[:],
		time.Duration(int(time.Millisecond)*writeLoopTO), io.MultiWriter(ws...),
		resReadCloser,
		func(s string) { errChan <- pe.New(s) },
	)
	for len(errChan) > 0 {
		err = pe.Join(err, <-errChan)
	}

	if t.responBuf != nil {
		t.Respon = t.responBuf.Bytes()
	}

	resReadCloser.Close()

	return
}

func (t *Req) Wait() (err error) {
	t.l.RLock()
	err = t.err
	t.l.RUnlock()
	return
}

func (t *Req) Close() { t.Cancel() }
func (t *Req) Cancel() {
	if p := t.cancelP.Load(); p != nil {
		(*p)()
	}
}

func (t *Req) IsLive() bool {
	return t.state.Load() == running
}

func (t *Req) prepareRes(ctx context.Context, val *Rval) (ctx1 context.Context, ctxf1 context.CancelFunc) {
	if !val.NoResponse {
		if t.responBuf == nil {
			t.responBuf = new(bytes.Buffer)
			t.Respon = t.responBuf.Bytes()
		} else {
			t.responBuf.Reset()
		}
	} else {
		t.Respon = []byte{}
		t.responBuf = nil
	}
	t.Response = nil
	t.err = nil

	if val.Timeout > 0 {
		ctx1, ctxf1 = context.WithTimeout(ctx, time.Duration(val.Timeout*int(time.Millisecond)))
	} else {
		ctx1, ctxf1 = context.WithCancel(ctx)
	}
	return
}

func (t *Req) prepare(val *Rval) (ctx context.Context, cancel context.CancelFunc) {
	t.UsedTime = 0
	t.responFile = nil
	t.callTree = ""
	for i := 2; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok {
			break
		} else {
			t.callTree += fmt.Sprintf("call by %s\n\t%s:%d\n", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	if cap(t.copyResBuf) == 0 {
		t.copyResBuf = make([]byte, 1<<17)
	}
	if val.Ctx != nil {
		ctx, cancel = context.WithCancel(val.Ctx)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	if val.SaveToPipe != nil {
		go func() {
			<-ctx.Done()
			if e := val.SaveToPipe.CloseWithError(context.Canceled); e != nil {
				println(e)
			}
		}()
	}
	if val.RawPipe != nil {
		go func() {
			<-ctx.Done()
			if e := val.RawPipe.ResCloseWithError(context.Canceled); e != nil {
				println(e)
			}
		}()
	}

	if val.Method == "" {
		if len(val.PostStr) > 0 {
			val.Method = "POST"
		} else if val.JustResponseCode {
			val.Method = "OPTIONS"
		} else {
			val.Method = "GET"
		}
	}

	return
}

func (t *Req) clean(val *Rval) {
	if t.responFile != nil {
		t.responFile.Close()
	}
	if val.SaveToPipe != nil {
		val.SaveToPipe.Close()
	}
	if val.RawPipe != nil {
		val.RawPipe.ReqClose()
		val.RawPipe.ResClose()
	}
}

func (t *Req) updateUseDur(u time.Time) {
	t.UsedTime = time.Since(u)
}

func IsTimeout(e error) bool {
	if errors.Is(e, context.DeadlineExceeded) {
		return true
	}
	if net_err, ok := e.(net.Error); ok && net_err.Timeout() {
		return true
	}
	if os.IsTimeout(e) {
		return true
	}
	return false
}

func IsCancel(e error) bool {
	return errors.Is(e, context.Canceled)
}

func ToForm(m map[string]string) (postStr string, contentType string) {
	var buf strings.Builder
	sign := s.Rand(0, 30)
	for k, v := range m {
		buf.WriteString(`-----------------------------` + sign + "\n")
		buf.WriteString(`Content-Disposition: form-data; name="` + k + `"` + "\n\n")
		buf.WriteString(v + "\n")
	}
	buf.WriteString(`-----------------------------` + sign + `--`)
	return buf.String(), `multipart/form-data; boundary=---------------------------` + sign
}

func ResDate(res *http.Response) (time.Time, error) {
	if date := res.Header.Get("date"); date != `` {
		return time.Parse(time.RFC1123, date)
	} else {
		return time.Time{}, ErrNoDate.New()
	}
}
