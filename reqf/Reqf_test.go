package part

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	compress "github.com/qydysky/part/compress"
	pio "github.com/qydysky/part/io"
	web "github.com/qydysky/part/web"
)

var addr = "127.0.0.1:10001"

var reuse = New()

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
		`/reply`: func(w http.ResponseWriter, r *http.Request) {
			io.Copy(w, r.Body)
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
		`/stream`: func(w http.ResponseWriter, r *http.Request) {
			flusher, flushSupport := w.(http.Flusher)
			if flushSupport {
				flusher.Flush()
			}
			for {
				select {
				case <-r.Context().Done():
					println("server req ctx done")
					return
				default:
					w.Write([]byte{'0'})
					flusher.Flush()
				}
			}
		},
		`/exit`: func(_ http.ResponseWriter, _ *http.Request) {
			s.Server.Shutdown(context.Background())
		},
	})
	time.Sleep(time.Second)
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/no",
	})
}

// go test -timeout 30s -run ^Test_reuse$ github.com/qydysky/part/reqf -race -count=1 -v -memprofile mem.out
func Test_reuse(t *testing.T) {
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/no",
	})
	if !bytes.Equal(reuse.Respon, []byte("abc强强强强")) {
		t.Fail()
	}
}

// 2710            430080 ns/op            9896 B/op        111 allocs/op
func Benchmark(b *testing.B) {
	rval := Rval{
		Url: "http://" + addr + "/no",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reuse.Reqf(rval)
		if !bytes.Equal(reuse.Respon, []byte("abc强强强强")) {
			b.Fail()
		}
	}
}

func Test14(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	i, o := io.Pipe()

	r := New()
	if e := r.Reqf(Rval{
		Url:         "http://" + addr + "/stream",
		Ctx:         ctx,
		NoResponse:  true,
		SaveToPipe:  pio.NewPipeRaw(i, o),
		Async:       true,
		WriteLoopTO: 5*1000*2 + 1,
	}); e != nil {
		t.Log(e)
	}

	start := time.Now()

	t.Log("Do", time.Since(start))

	go func() {
		buf := make([]byte, 1<<8)
		for {
			if n, e := i.Read(buf); n != 0 {
				if time.Since(start) > time.Second {
					cancel()
					t.Log("Cancel", time.Since(start))
					break
				}
				// do nothing
				continue
			} else if e != nil {
				t.Log(e)
				break
			}
		}
	}()

	if !errors.Is(r.Wait(), context.Canceled) {
		t.Fatal()
	}
	t.Log("Do finished", time.Since(start))
}

func Test_req13(t *testing.T) {
	r := New()
	e := r.Reqf(Rval{
		Url:     "http://" + addr + "/code?code=403",
		Timeout: 1000,
		Retry:   2,
	})
	if e.Error() != "403 Forbidden" {
		t.Fatal(e.Error())
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

// func Test_req5(t *testing.T) {
// 	r := New()
// 	{
// 		c := make(chan []byte)
// 		r.Reqf(Rval{
// 			Url:        "http://" + addr + "/to",
// 			Timeout:    1000,
// 			Async:      true,
// 			SaveToChan: c,
// 		})
// 		for {
// 			buf := <-c
// 			if len(buf) == 0 {
// 				break
// 			}
// 		}
// 		if !IsTimeout(r.Wait()) {
// 			t.Error("async IsTimeout fail")
// 		}
// 	}
// }

// func Test_req6(t *testing.T) {
// 	r := New()
// 	{
// 		c := make(chan []byte)
// 		r.Reqf(Rval{
// 			Url:        "http://" + addr + "/no",
// 			Async:      true,
// 			SaveToChan: c,
// 		})
// 		b := []byte{}
// 		for {
// 			buf := <-c
// 			if len(buf) == 0 {
// 				break
// 			}
// 			b = append(b, buf...)
// 		}
// 		if !bytes.Equal(b, []byte("abc强强强强")) {
// 			t.Error("chan fail")
// 		}
// 	}
// }

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
			Url:        "http://" + addr + "/1min",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			Async:      true,
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
			_, _ = rc.Read(buf)
			time.Sleep(time.Millisecond * 500)
			r.Cancel()
		}()
		r.Reqf(Rval{
			Url:        "http://" + addr + "/1min",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			Async:      true,
		})
		if !IsCancel(r.Wait()) {
			t.Fatal("read from block response")
		}
	}
}

// panic
/*
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
			SaveToPipe: wc,
			Async:            true,
		})
		if !IsCancel(r.Wait()) {
			t.Fatal("write to block io.pipe")
		}
	}
}
*/

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
			Url:        "http://" + addr + "/br",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			Async:      true,
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
			Url:        "http://" + addr + "/gzip",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			Async:      true,
		})
		<-c
	}
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			d, _ := io.ReadAll(rc)
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("flate fail", d)
			}
			close(c)
		}()
		if e := r.Reqf(Rval{
			Url:        "http://" + addr + "/flate",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
		}); e != nil {
			t.Error(e)
		}
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
			Url:        "http://" + addr + "/flate",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			Async:      true,
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
		var wg sync.WaitGroup
		rc, wc := io.Pipe()
		wg.Add(1)
		go func() {
			var buf []byte = make([]byte, 1<<16)
			n, _ := rc.Read(buf)
			d := buf[:n]
			if !bytes.Equal(d, []byte("abc强强强强")) {
				t.Error("io async fail", d)
			}
			wg.Done()
		}()
		r.Reqf(Rval{
			Url:        "http://" + addr + "/flate",
			SaveToPipe: pio.NewPipeRaw(rc, wc),
			NoResponse: true,
			Async:      true,
		})
		r.Wait()
		if len(r.Respon) != 0 {
			t.Error("io async fail", r.Respon)
		}
		wg.Wait()
	}
}

func Test_req5(t *testing.T) {
	r := New()
	r.Reqf(Rval{
		Url:     "http://" + addr + "/reply",
		PostStr: "123",
	})
	if !bytes.Equal(r.Respon, []byte("123")) {
		t.Fatal()
	}

	raw := NewRawReqRes()
	buf := []byte("123")
	r.Reqf(Rval{
		Url:        "http://" + addr + "/reply",
		Async:      true,
		RawPipe:    raw,
		NoResponse: true,
	})
	if _, e := raw.ReqWrite(buf); e != nil {
		t.Fatal(e)
	}
	raw.ReqClose()
	clear(buf)
	if _, e := raw.ResRead(buf); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(e)
	}
	if !bytes.Equal([]byte("123"), buf) {
		t.Log(r.Respon, buf)
		t.Fatal()
	}
	if _, e := ResDate(r.Response); e != nil {
		t.Fatal()
	}
}
