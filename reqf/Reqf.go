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

	br "github.com/andybalholm/brotli"
	pio "github.com/qydysky/part/io"
	s "github.com/qydysky/part/strings"
	// "encoding/binary"
)

type Rval struct {
	Url              string
	PostStr          string
	Timeout          int
	Proxy            string
	Retry            int
	SleepTime        int
	WriteLoopTO      int
	JustResponseCode bool
	NoResponse       bool
	// 当Async为true时，Respon、Response必须在Wait()之后读取，否则有DATA RACE可能
	Async   bool
	Cookies []*http.Cookie
	Ctx     context.Context

	SaveToPath       string
	SaveToChan       chan []byte
	SaveToPipeWriter *io.PipeWriter

	Header map[string]string
}

const (
	defaultUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"
	defaultAccept = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`
	free          = iota
	running
)

var (
	ErrEmptyUrl         = errors.New("ErrEmptyUrl")
	ErrNewRequest       = errors.New("ErrNewRequest")
	ErrClientDo         = errors.New("ErrClientDo")
	ErrResponFileCreate = errors.New("ErrResponFileCreate")
	ErrWriteRes         = errors.New("ErrWriteRes")
	ErrReadRes          = errors.New("ErrReadRes")
)

type Req struct {
	// 当Async为true时，必须在Wait()之后读取，否则有DATA RACE可能
	Respon []byte
	// 当Async为true时，必须在Wait()之后读取，否则有DATA RACE可能
	Response *http.Response
	UsedTime time.Duration

	cancelP atomic.Pointer[func()]
	ctx     context.Context
	state   atomic.Int32

	responFile *os.File
	responBuf  *bytes.Buffer
	err        error
	callTree   string

	l sync.RWMutex
}

func New() *Req {
	return new(Req)
}

func (t *Req) Reqf(val Rval) error {
	t.l.Lock()
	t.state.Store(running)

	t.prepare(&val)

	// 同步
	if !val.Async {
		beginTime := time.Now()

		for i := 0; i <= val.Retry; i++ {
			ctx, cancle := t.prepareRes(&val)
			t.err = t.Reqf_1(ctx, val)
			if cancle != nil {
				cancle()
			}
			if t.err == nil || IsCancel(t.err) {
				break
			}
			if val.SleepTime != 0 {
				time.Sleep(time.Duration(val.SleepTime * int(time.Millisecond)))
			}
		}

		t.updateUseDur(beginTime)
		t.clean(&val)
		t.state.Store(free)
		t.l.Unlock()
		return t.err
	}

	//异步
	go func() {
		beginTime := time.Now()

		for i := 0; i <= val.Retry; i++ {
			ctx, cancle := t.prepareRes(&val)
			t.err = t.Reqf_1(ctx, val)
			if cancle != nil {
				cancle()
			}
			if t.err == nil || IsCancel(t.err) {
				break
			}
			if val.SleepTime != 0 {
				time.Sleep(time.Duration(val.SleepTime * int(time.Millisecond)))
			}
		}

		t.updateUseDur(beginTime)
		t.clean(&val)
		t.state.Store(free)
		t.l.Unlock()
	}()
	return nil
}

func (t *Req) Reqf_1(ctx context.Context, val Rval) (err error) {
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
		return ErrEmptyUrl
	}

	Method := "GET"
	var body io.Reader
	if len(val.PostStr) > 0 {
		Method = "POST"
		body = strings.NewReader(val.PostStr)
		if _, ok := Header["Content-Type"]; !ok {
			Header["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	req, e := http.NewRequestWithContext(ctx, Method, val.Url, body)
	if e != nil {
		return errors.Join(ErrNewRequest, e)
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
	if val.SaveToPath != "" || val.SaveToPipeWriter != nil {
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
		return errors.Join(ErrClientDo, e)
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
			return errors.Join(err, ErrResponFileCreate, e)
		}
		ws = append(ws, t.responFile)
	}
	if val.SaveToPipeWriter != nil {
		ws = append(ws, val.SaveToPipeWriter)
	}
	if !val.NoResponse {
		//will clear t.Respon
		t.responBuf.Reset()
		ws = append(ws, t.responBuf)
	}

	w := io.MultiWriter(ws...)

	var resReadCloser io.ReadCloser
	if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 {
		switch compress_type[0] {
		case `br`:
			resReadCloser = pio.RWC{R: br.NewReader(resp.Body).Read}
		case `gzip`:
			resReadCloser, _ = gzip.NewReader(resp.Body)
		case `deflate`:
			resReadCloser = flate.NewReader(resp.Body)
		default:
			resReadCloser = resp.Body
		}
	} else {
		resReadCloser = resp.Body
	}

	writeLoopTO := val.WriteLoopTO
	if writeLoopTO == 0 {
		writeLoopTO = 1000
	}
	rwc := pio.WithCtxTO(ctx, t.callTree, time.Duration(int(time.Millisecond)*writeLoopTO), w, resReadCloser)
	defer rwc.Close()

	for buf := make([]byte, 2048); true; {
		if n, e := rwc.Read(buf); n != 0 {
			if wn, we := rwc.Write(buf[:n]); wn != 0 {
				if val.SaveToChan != nil {
					val.SaveToChan <- buf[:n]
				}
			} else if we != nil {
				if !errors.Is(e, io.EOF) {
					err = errors.Join(err, ErrWriteRes, e)
				}
				break
			}
		} else if e != nil {
			if !errors.Is(e, io.EOF) {
				err = errors.Join(err, ErrReadRes, e)
			}
			break
		}
	}

	resp.Body.Close()

	if t.responBuf != nil {
		t.Respon = t.responBuf.Bytes()
	}

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

func (t *Req) prepareRes(val *Rval) (context.Context, context.CancelFunc) {
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
		return context.WithTimeout(t.ctx, time.Duration(val.Timeout*int(time.Millisecond)))
	}
	return t.ctx, nil
}

func (t *Req) prepare(val *Rval) {
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
	var cancel func()
	if val.Ctx != nil {
		t.ctx, cancel = context.WithCancel(val.Ctx)
	} else {
		t.ctx, cancel = context.WithCancel(context.Background())
	}
	t.cancelP.Store(&cancel)
}

func (t *Req) clean(val *Rval) {
	if p := t.cancelP.Load(); p != nil {
		(*p)()
	}
	if val.SaveToChan != nil {
		close(val.SaveToChan)
	}
	if t.responFile != nil {
		t.responFile.Close()
	}
	if val.SaveToPipeWriter != nil {
		val.SaveToPipeWriter.Close()
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
