package part

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	compress "github.com/qydysky/part/compress"
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
	Cookies          []*http.Cookie

	SaveToPath       string
	SaveToChan       chan []byte // deprecated
	SaveToPipeWriter *io.PipeWriter

	Header map[string]string
}

type Req struct {
	Respon   []byte
	Response *http.Response
	UsedTime time.Duration

	cancel *signal.Signal
	sync.Mutex
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
	t.Lock()
	defer t.Unlock()

	t.Respon = []byte{}
	t.Response = nil
	t.UsedTime = 0

	var returnErr error

	_val := val

	t.cancel = signal.Init()
	defer t.cancel.Done()

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
	req, e := http.NewRequest(Method, val.Url, body)
	if e != nil {
		panic(e)
	}
	req = req.WithContext(cx)

	var done = make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-t.cancel.WaitC():
			cancel()
		case <-done:
		}
	}()

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
	defer resp.Body.Close()
	defer func() {
		t.UsedTime = time.Since(beginTime)
	}()

	if val.JustResponseCode {
		return
	}

	if resp.StatusCode >= 400 {
		err = errors.New(strconv.Itoa(resp.StatusCode))
	}

	if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 && (compress_type[0] == `br` ||
		compress_type[0] == `gzip` ||
		compress_type[0] == `deflate`) {

		if val.NoResponse {
			return errors.New("respose had compress, must load all data, but NoResponse is true")
		}

		var err error
		t.Respon, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 {
			switch compress_type[0] {
			case `br`:
				if tmp, err := compress.UnBr(t.Respon); err != nil {
					return err
				} else {
					t.Respon = append([]byte{}, tmp...)
				}
			case `gzip`:
				if tmp, err := compress.UnGzip(t.Respon); err != nil {
					return err
				} else {
					t.Respon = append([]byte{}, tmp...)
				}
			case `deflate`:
				if tmp, err := compress.UnFlate(t.Respon); err != nil {
					return err
				} else {
					t.Respon = append([]byte{}, tmp...)
				}
			default:
			}
		}
	} else {
		var ws []io.Writer
		if val.SaveToPath != "" {
			out, err := os.Create(val.SaveToPath)
			if err != nil {
				out.Close()
				return err
			}
			defer out.Close()
			ws = append(ws, out)
		}
		if val.SaveToPipeWriter != nil {
			defer val.SaveToPipeWriter.Close()
			ws = append(ws, val.SaveToPipeWriter)
		}
		// if val.SaveToChan != nil {
		// 	r, w := io.Pipe()
		// 	go func() {
		// 		buf := make([]byte, 1<<16)
		// 		for {
		// 			n, e := r.Read(buf)
		// 			if n != 0 {
		// 				val.SaveToChan <- buf[:n]
		// 			} else if e != nil {
		// 				defer close(val.SaveToChan)
		// 				break
		// 			}
		// 		}
		// 	}()
		// 	defer w.Close()
		// 	ws = append(ws, w)
		// }
		if !val.NoResponse {
			var buf bytes.Buffer
			defer func() {
				t.Respon = buf.Bytes()
			}()
			ws = append(ws, &buf)
		}

		w := io.MultiWriter(ws...)
		s := signal.Init()
		go func() {
			buf := make([]byte, 1<<16)
			for {
				if n, e := resp.Body.Read(buf); n != 0 {
					w.Write(buf[:n])
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
			s.Done()
		}()
		s.Wait()
		// if _, e := io.Copy(w, resp.Body); e != nil {
		// 	err = e
		// }
	}
	return
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
