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
	"strings"
	"sync"
	"time"

	flate "compress/flate"
	gzip "compress/gzip"

	br "github.com/andybalholm/brotli"
	signal "github.com/qydysky/part/signal"
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
	JustResponseCode bool
	NoResponse       bool
	Async            bool
	Cookies          []*http.Cookie

	SaveToPath       string
	SaveToChan       chan []byte
	SaveToPipeWriter *io.PipeWriter

	Header map[string]string
}

type Req struct {
	Respon    []byte
	responBuf *bytes.Buffer
	Response  *http.Response
	UsedTime  time.Duration

	cancelFs chan func()
	cancel   *signal.Signal
	running  *signal.Signal
	isLive   *signal.Signal

	responFile *os.File
	err        error

	init sync.RWMutex
	l    sync.Mutex
}

func New() *Req {
	return new(Req)
}

func (t *Req) Reqf(val Rval) error {
	t.isLive.Wait()
	t.l.Lock()
	t.init.Lock()

	t.Respon = t.Respon[:0]
	if t.responBuf == nil {
		t.responBuf = new(bytes.Buffer)
	}
	t.Response = nil
	t.UsedTime = 0
	t.cancelFs = make(chan func(), 5)
	t.isLive = signal.Init()
	t.cancel = signal.Init()
	t.running = signal.Init()
	t.responFile = nil
	t.err = nil
	go func() {
		cancel, cancelFin := t.cancel.WaitC()
		defer cancelFin()
		running, runningFin := t.running.WaitC()
		defer runningFin()

		select {
		case <-cancel:
			for len(t.cancelFs) != 0 {
				(<-t.cancelFs)()
			}
		case <-running:
		}
	}()

	t.init.Unlock()

	go func() {
		beginTime := time.Now()
		_val := val

		for SleepTime, Retry := _val.SleepTime, _val.Retry; Retry >= 0; Retry -= 1 {
			for len(t.cancelFs) != 0 {
				<-t.cancelFs
			}
			t.err = t.Reqf_1(_val)
			if t.err == nil || IsCancel(t.err) {
				break
			}
			time.Sleep(time.Duration(SleepTime) * time.Millisecond)
		}
		t.UsedTime = time.Since(beginTime)
		t.running.Done()
	}()

	if !val.Async {
		t.Wait()
		if val.SaveToChan != nil {
			close(val.SaveToChan)
		}
		if t.responFile != nil {
			t.responFile.Close()
		}
		if val.SaveToPipeWriter != nil {
			val.SaveToPipeWriter.Close()
		}
		t.cancel.Done()
		t.running.Done()
		t.l.Unlock()
		t.isLive.Done()
		return t.err
	} else {
		go func() {
			t.Wait()
			if val.SaveToChan != nil {
				close(val.SaveToChan)
			}
			if t.responFile != nil {
				t.responFile.Close()
			}
			if val.SaveToPipeWriter != nil {
				val.SaveToPipeWriter.Close()
			}
			t.cancel.Done()
			t.running.Done()
			t.l.Unlock()
			t.isLive.Done()
		}()
	}
	return nil
}

func (t *Req) Reqf_1(val Rval) (err error) {
	var (
		Header map[string]string = val.Header
	)

	var client http.Client

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
		return errors.New("url is empty")
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

	cx, cancel := context.WithCancel(context.Background())
	if val.Timeout > 0 {
		cx, cancel = context.WithTimeout(cx, time.Duration(val.Timeout)*time.Millisecond)
	}
	t.cancelFs <- cancel

	req, e := http.NewRequest(Method, val.Url, body)
	if e != nil {
		panic(e)
	}
	req = req.WithContext(cx)

	for _, v := range val.Cookies {
		req.AddCookie(v)
	}

	if _, ok := Header["Accept"]; !ok {
		Header["Accept"] = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`
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
		Header["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36"
	}

	for k, v := range Header {
		req.Header.Set(k, v)
	}

	if !t.cancel.Islive() {
		err = context.Canceled
		return
	}

	resp, e := client.Do(req)

	if e != nil {
		err = e
		return
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
		if err != nil {
			t.responFile.Close()
			err = e
			return
		}
		ws = append(ws, t.responFile)
		t.cancelFs <- func() { t.responFile.Close() }
	}
	if val.SaveToPipeWriter != nil {
		ws = append(ws, val.SaveToPipeWriter)
		t.cancelFs <- func() { val.SaveToPipeWriter.Close() }
	}
	if !val.NoResponse {
		t.responBuf.Reset()
		ws = append(ws, t.responBuf)
	}

	w := io.MultiWriter(ws...)

	var resReader io.Reader
	if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 {
		switch compress_type[0] {
		case `br`:
			resReader = br.NewReader(resp.Body)
		case `gzip`:
			resReader, _ = gzip.NewReader(resp.Body)
		case `deflate`:
			resReader = flate.NewReader(resp.Body)
		default:
			resReader = resp.Body
		}
	} else {
		resReader = resp.Body
	}
	t.cancelFs <- func() { resp.Body.Close() }

	buf := make([]byte, 512)

	for {
		if n, e := resReader.Read(buf); n != 0 {
			w.Write(buf[:n])
			if val.SaveToChan != nil {
				val.SaveToChan <- buf[:n]
			}
		} else if e != nil {
			if !errors.Is(e, io.EOF) {
				err = e
			}
			break
		}

		if !t.cancel.Islive() {
			err = context.Canceled
			break
		}
	}

	resp.Body.Close()

	if t.responBuf != nil {
		t.Respon = t.responBuf.Bytes()
	}

	return
}

func (t *Req) Wait() error {
	t.init.RLock()
	defer t.init.RUnlock()

	t.running.Wait()
	return t.err
}

func (t *Req) Cancel() { t.Close() }

func (t *Req) Close() {
	t.init.RLock()
	defer t.init.RUnlock()

	if !t.cancel.Islive() {
		return
	}
	t.cancel.Done()
}

func (t *Req) IsLive() bool {
	t.init.RLock()
	defer t.init.RUnlock()

	return t.isLive.Islive()
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
