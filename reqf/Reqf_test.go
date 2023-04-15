package part

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	compress "github.com/qydysky/part/compress"
	web "github.com/qydysky/part/web"
)

var addr = "127.0.0.1:10001"

func init() {
	s := web.New(&http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * time.Duration(10),
	})
	s.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/code`: func(w http.ResponseWriter, r *http.Request) {
			code, _ := strconv.Atoi(r.URL.Query().Get(`code`))
			w.WriteHeader(code)
		},
		`/no`: func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("abc强强强强"))
		},
		`/br`: func(w http.ResponseWriter, _ *http.Request) {
			d, _ := compress.InBr([]byte("abc强强强强"), 6)
			w.Header().Set("Content-Encoding", "br")
			w.Write(d)
		},
		`/flate`: func(w http.ResponseWriter, _ *http.Request) {
			d, _ := compress.InFlate([]byte("abc强强强强"), -1)
			w.Header().Set("Content-Encoding", "deflate")
			w.Write(d)
		},
		`/gzip`: func(w http.ResponseWriter, _ *http.Request) {
			d, _ := compress.InGzip([]byte("abc强强强强"), -1)
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(d)
		},
		`/to`: func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(time.Minute)
			d, _ := compress.InGzip([]byte("abc强强强强"), -1)
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(d)
		},
		`/1min`: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			flusher, flushSupport := w.(http.Flusher)
			if flushSupport {
				flusher.Flush()
			}
			for i := 0; i < 3; i++ {
				w.Write([]byte("0"))
				if flushSupport {
					flusher.Flush()
				}
				time.Sleep(time.Second)
			}
		},
		`/json`: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			flusher, flushSupport := w.(http.Flusher)
			if flushSupport {
				flusher.Flush()
			}
			w.Write([]byte("{\"a\":"))
			if flushSupport {
				flusher.Flush()
			}
			time.Sleep(time.Millisecond * 20)
			w.Write([]byte("123}"))
			if flushSupport {
				flusher.Flush()
			}
		},
		`/exit`: func(_ http.ResponseWriter, _ *http.Request) {
			s.Server.Shutdown(context.Background())
		},
	})
	time.Sleep(time.Second)
}

func Test_req13(t *testing.T) {
	r := New()
	e := r.Reqf(Rval{
		Url:     "http://" + addr + "/code?code=403",
		Timeout: 1000,
		Retry:   2,
	})
	if e.Error() != "403 Forbidden" {
		t.Fatal()
	}
}

func Test_req7(t *testing.T) {
	r := New()
	r.Reqf(Rval{
		Url:   "http://" + addr + "/to",
		Async: true,
	})
	r.Cancel()
	if !IsCancel(r.Wait()) {
		t.Error("async Cancel fail")
	}
}

func Test_req(t *testing.T) {
	r := New()
	r.Reqf(Rval{
		Url: "http://" + addr + "/br",
	})
	if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
		t.Error("br fail", r.Respon)
	}
	r.Reqf(Rval{
		Url: "http://" + addr + "/gzip",
	})
	if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
		t.Error("gzip fail")
	}
	r.Reqf(Rval{
		Url: "http://" + addr + "/flate",
	})
	if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
		t.Error("flate fail")
	}
}

type J struct {
	A int `json:"a"`
}

func Test_req12(t *testing.T) {
	r := New()
	r.Reqf(Rval{
		Url:     "http://" + addr + "/json",
		Timeout: 10 * 1000,
		Retry:   2,
	})
	var j J
	json.Unmarshal(r.Respon, &j)
}

func Test_req2(t *testing.T) {
	r := New()
	{
		e := r.Reqf(Rval{
			Url:     "http://" + addr + "/to",
			Timeout: 1000,
		})
		if !IsTimeout(e) {
			t.Error("Timeout fail")
		}
	}
}

func Test_req4(t *testing.T) {
	r := New()
	{
		r.Reqf(Rval{
			Url:     "http://" + addr + "/to",
			Timeout: 1000,
			Async:   true,
		})
		if e := r.Wait(); !IsTimeout(e) {
			t.Error("Async Timeout fail", e)
		}
	}
}

func Test_req5(t *testing.T) {
	r := New()
	{
		c := make(chan []byte)
		r.Reqf(Rval{
			Url:        "http://" + addr + "/to",
			Timeout:    1000,
			Async:      true,
			SaveToChan: c,
		})
		for {
			buf := <-c
			if len(buf) == 0 {
				break
			}
		}
		if !IsTimeout(r.Wait()) {
			t.Error("async IsTimeout fail")
		}
	}
}

func Test_req6(t *testing.T) {
	r := New()
	{
		c := make(chan []byte)
		r.Reqf(Rval{
			Url:        "http://" + addr + "/no",
			Async:      true,
			SaveToChan: c,
		})
		b := []byte{}
		for {
			buf := <-c
			if len(buf) == 0 {
				break
			}
			b = append(b, buf...)
		}
		if !bytes.Equal(b, []byte("abc强强强强")) {
			t.Error("chan fail")
		}
	}
}

func Test_req11(t *testing.T) {
	r := New()
	{
		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			r.Cancel()
		}()
		e := r.Reqf(Rval{
			Url: "http://" + addr + "/to",
		})
		if !IsCancel(e) {
			t.Error("Cancel fail")
		}
	}
}

func Test_req9(t *testing.T) {
	r := New()
	{
		rc, wc := io.Pipe()
		go func() {
			var buf []byte = make([]byte, 1<<16)
			for {
				n, _ := rc.Read(buf)
				if n == 0 {
					break
				}
			}
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/1min",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		if r.Wait() != nil {
			t.Fatal()
		}
	}
}

func Test_req8(t *testing.T) {
	r := New()
	{
		rc, wc := io.Pipe()
		go func() {
			var buf []byte = make([]byte, 1<<16)
			rc.Read(buf)
			time.Sleep(time.Millisecond * 500)
			r.Cancel()
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/1min",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		if !IsCancel(r.Wait()) {
			t.Fatal("read from block response")
		}
	}
}

func Test_req10(t *testing.T) {
	r := New()
	{
		_, wc := io.Pipe()
		go func() {
			time.Sleep(time.Millisecond * 500)
			r.Cancel()
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/1min",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		if !IsCancel(r.Wait()) {
			t.Fatal("write to block io.pipe")
		}
	}
}

func Test_req3(t *testing.T) {
	r := New()
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			d, _ := io.ReadAll(rc)
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("br fail")
			}
			close(c)
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/br",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		<-c
	}
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			d, _ := io.ReadAll(rc)
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("gzip fail")
			}
			close(c)
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/gzip",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		<-c
	}
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			d, _ := io.ReadAll(rc)
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("flate fail")
			}
			close(c)
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/flate",
			SaveToPipeWriter: wc,
		})
		<-c
	}
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			var buf []byte = make([]byte, 1<<16)
			n, _ := rc.Read(buf)
			d := buf[:n]
			// d, _ := io.ReadAll(rc)
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("flate fail")
			}
			close(c)
		}()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/flate",
			SaveToPipeWriter: wc,
			Async:            true,
		})
		<-c
	}
	{
		r.Reqf(Rval{
			Url:   "http://" + addr + "/flate",
			Async: true,
		})
		r.Wait()
		if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
			t.Error("async fail", r.Respon)
		}
	}
	{
		rc, wc := io.Pipe()
		r.Reqf(Rval{
			Url:              "http://" + addr + "/flate",
			SaveToPipeWriter: wc,
			NoResponse:       true,
			Async:            true,
		})
		if len(r.Respon) != 0 {
			t.Error("io async fail")
		}
		var buf []byte = make([]byte, 1<<16)
		n, _ := rc.Read(buf)
		d := buf[:n]
		if !bytes.Equal(d, []byte("abc强强强强")) {
			t.Error("io async fail", d)
		}
	}
}
