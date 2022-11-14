package part

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	ReadTimeout      int // deprecated
	ConnectTimeout   int //	deprecated
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
	Respon   []byte
	Response *http.Response
	UsedTime time.Duration

	cancel     *signal.Signal
	running    *signal.Signal
	responBuf  *bytes.Buffer
	responFile *os.File
	asyncErr   error
}

func New() *Req {
	return new(Req)
}

// func main(){
//     var _ReqfVal = ReqfVal{
//         Url:url,
//         Proxy:proxy,
// 		Timeout:10,
// 		Retry:2,
//     }
//     Reqf(_ReqfVal)
// }

func (t *Req) Reqf(val Rval) error {

	if val.SaveToChan != nil && len(val.SaveToChan) == 1 && !val.Async {
		panic("must make sure chan size larger then 1 or use Async true")
	}
	if val.SaveToPipeWriter != nil && !val.Async {
		panic("SaveToPipeWriter must use Async true")
	}

	t.Respon = []byte{}
	t.Response = nil
	t.UsedTime = 0
	t.cancel = signal.Init()
	t.running = signal.Init()

	var returnErr error

	_val := val

	for SleepTime, Retry := _val.SleepTime, _val.Retry; Retry >= 0; Retry -= 1 {
		returnErr = t.Reqf_1(_val)
		select {
		case <-t.cancel.WaitC(): //cancel
			return returnErr
		default:
			if returnErr == nil {
				return nil
			}
		}
		time.Sleep(time.Duration(SleepTime) * time.Millisecond)
	}

	if !val.Async || returnErr != nil {
		t.asyncErr = returnErr
		if val.SaveToChan != nil {
			close(val.SaveToChan)
		}
		if t.responFile != nil {
			t.responFile.Close()
		}
		if val.SaveToPipeWriter != nil {
			val.SaveToPipeWriter.Close()
		}
		if t.responBuf != nil {
			t.Respon = t.responBuf.Bytes()
		}
		t.running.Done()
		t.cancel.Done()
	}
	return returnErr
}

func (t *Req) Reqf_1(val Rval) (err error) {
	var (
		Header map[string]string = val.Header
	)

	var beginTime time.Time = time.Now()

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

	go func() {
		select {
		case <-t.cancel.WaitC():
			cancel()
		case <-t.running.WaitC():
		}
	}()

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

	resp, err := client.Do(req)
	if v, ok := Header["Connection"]; ok && strings.ToLower(v) != "keep-alive" {
		defer client.CloseIdleConnections()
	}

	if err != nil {
		return err
	}

	t.Response = resp
	defer func() {
		t.UsedTime = time.Since(beginTime)
	}()

	if val.JustResponseCode {
		return
	}

	if resp.StatusCode >= 400 {
		err = errors.New(strconv.Itoa(resp.StatusCode))
	}

	var ws []io.Writer
	if val.SaveToPath != "" {
		t.responFile, err = os.Create(val.SaveToPath)
		if err != nil {
			t.responFile.Close()
			return err
		}
		ws = append(ws, t.responFile)
	}
	if val.SaveToPipeWriter != nil {
		ws = append(ws, val.SaveToPipeWriter)
	}
	if !val.NoResponse {
		if t.responBuf == nil {
			t.responBuf = new(bytes.Buffer)
		} else {
			t.responBuf.Reset()
		}
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

	go func() {
		buf := make([]byte, 512)

		for {
			if n, e := resReader.Read(buf); n != 0 {
				w.Write(buf[:n])
				select {
				case val.SaveToChan <- buf[:n]:
				default:
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

		if val.Async {
			t.asyncErr = err
		}
		resp.Body.Close()
		if val.SaveToChan != nil {
			close(val.SaveToChan)
		}
		if t.responFile != nil {
			t.responFile.Close()
		}
		if val.SaveToPipeWriter != nil {
			val.SaveToPipeWriter.Close()
		}
		if t.responBuf != nil {
			t.Respon = t.responBuf.Bytes()
		}
		t.running.Done()
	}()
	if !val.Async {
		t.Wait()
	}
	// if _, e := io.Copy(w, resp.Body); e != nil {
	// 	err = e
	// }
	return
}

func (t *Req) Wait() error {
	t.running.Wait()
	return t.asyncErr
}

func (t *Req) Cancel() { t.Close() }

func (t *Req) Close() {
	if !t.cancel.Islive() {
		return
	}
	t.cancel.Done()
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
