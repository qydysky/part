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
	"strings"
	"sync"
	"time"

	compress "github.com/qydysky/part/compress"
	pio "github.com/qydysky/part/io"
	signal "github.com/qydysky/part/signal"
	// "encoding/binary"
)

var (
	ErrConnectTimeout = errors.New("ErrConnectTimeout")
	ErrReadTimeout    = errors.New("ErrReadTimeout")
)

type Rval struct {
	Url              string
	PostStr          string
	Timeout          int
	ReadTimeout      int
	ConnectTimeout   int
	Proxy            string
	Retry            int
	SleepTime        int
	JustResponseCode bool
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

	var returnErr error

	_val := val

	t.cancel = signal.Init()

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
		Url              string         = val.Url
		PostStr          string         = val.PostStr
		Proxy            string         = val.Proxy
		Timeout          int            = val.Timeout
		ReadTimeout      int            = val.ReadTimeout
		ConnectTimeout   int            = val.ConnectTimeout
		JustResponseCode bool           = val.JustResponseCode
		SaveToChan       chan []byte    = val.SaveToChan
		SaveToPath       string         = val.SaveToPath
		SaveToPipeWriter *io.PipeWriter = val.SaveToPipeWriter

		Header map[string]string = val.Header
	)

	var beginTime time.Time = time.Now()

	var client http.Client

	if Header == nil {
		Header = make(map[string]string)
	}

	if Proxy != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(Proxy)
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

	if Url == "" {
		return errors.New("url is empty")
	}

	Method := "GET"
	var body io.Reader
	if len(PostStr) > 0 {
		Method = "POST"
		body = strings.NewReader(PostStr)
		if _, ok := Header["Content-Type"]; !ok {
			Header["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	cx, cancel := context.WithCancel(context.Background())
	if Timeout > 0 {
		cx, cancel = context.WithTimeout(cx, time.Duration(Timeout)*time.Millisecond)
	}
	req, e := http.NewRequest(Method, Url, body)
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
	if SaveToPath != "" {
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

	var (
		saveToFile func(io.Reader, string) ([]byte, error) = func(Body io.Reader, filepath string) (bodyB []byte, err error) {
			out, err := os.Create(filepath + ".dtmp")
			if err != nil {
				out.Close()
				return bodyB, err
			}
			var buf bytes.Buffer

			w := io.MultiWriter(out, &buf)
			if _, err = io.Copy(w, Body); err != nil {
				out.Close()
				return bodyB, err
			}
			out.Close()
			bodyB = buf.Bytes()

			if err = os.RemoveAll(filepath); err != nil {
				return bodyB, err
			}
			if err = os.Rename(filepath+".dtmp", filepath); err != nil {
				return bodyB, err
			}
			return bodyB, nil
		}
	)
	t.Response = resp
	if !JustResponseCode {
		defer resp.Body.Close()
		if compress_type := resp.Header[`Content-Encoding`]; len(compress_type) != 0 && (compress_type[0] == `br` ||
			compress_type[0] == `gzip` ||
			compress_type[0] == `deflate`) {
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
			if SaveToPath != "" {
				if bodyB, err := saveToFile(resp.Body, SaveToPath); err != nil {
					return err
				} else {
					if len(bodyB) != 0 {
						if SaveToChan != nil {
							SaveToChan <- bodyB
						} else if SaveToPipeWriter != nil {
							SaveToPipeWriter.Write(bodyB)
						} else {
							t.Respon = append(t.Respon, bodyB...)
						}
					} else {
						return io.EOF
					}

					if SaveToChan != nil {
						close(SaveToChan)
					}
					if SaveToPipeWriter != nil {
						SaveToPipeWriter.Close()
					}
				}
			} else {
				rc, _ := pio.RW2Chan(resp.Body, nil)
				var After = func(ReadTimeout int) (c <-chan time.Time) {
					if ReadTimeout > 0 {
						c = time.NewTimer(time.Millisecond * time.Duration(ReadTimeout)).C
					}
					return
				}

				select {
				case buf := <-rc:
					if len(buf) != 0 {
						if SaveToChan != nil {
							SaveToChan <- buf
						} else if SaveToPipeWriter != nil {
							SaveToPipeWriter.Write(buf)
						} else {
							t.Respon = append(t.Respon, buf...)
						}
					} else {
						err = io.EOF
						return
					}
				case <-After(ConnectTimeout):
					err = ErrConnectTimeout
					return
				}

				for loop := true; loop; {
					select {
					case buf := <-rc:
						if len(buf) != 0 {
							if SaveToChan != nil {
								SaveToChan <- buf
							} else if SaveToPipeWriter != nil {
								SaveToPipeWriter.Write(buf)
							} else {
								t.Respon = append(t.Respon, buf...)
							}
						} else {
							err = io.EOF
							loop = false
						}
					case <-After(ReadTimeout):
						err = ErrReadTimeout
						loop = false
					}
					if !t.cancel.Islive() {
						err = context.Canceled
						loop = false
						break
					}
				}
				if SaveToChan != nil {
					close(SaveToChan)
				}
				if SaveToPipeWriter != nil {
					SaveToPipeWriter.Close()
				}
			}
		}
	} else {
		resp.Body.Close()
	}

	t.UsedTime = time.Since(beginTime)

	return nil
}

func (t *Req) Cancel() { t.Close() }

func (t *Req) Close() {
	if !t.cancel.Islive() {
		return
	}
	t.cancel.Done()
}

func IsTimeout(e error) bool {
	if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, ErrConnectTimeout) || errors.Is(e, ErrReadTimeout) {
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
