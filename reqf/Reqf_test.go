package part

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
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
					time.Sleep(time.Millisecond * 500)
				}
			}
		},
		`/exit`: func(_ http.ResponseWriter, _ *http.Request) {
			s.Server.Shutdown(context.Background())
		},
		`/header`: func(w http.ResponseWriter, r *http.Request) {
			for k, v := range r.Header {
				w.Header().Set(k, v[0])
			}
		},
		`/nores`: func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		},
	})
	time.Sleep(time.Second)
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/no",
	})
}

// https 10879 B/op        141 allocs/op
// http  9560 B/op        106 allocs/op
func Benchmark_1(b *testing.B) {
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/reply",
	})
	for b.Loop() {
		reuse.Reqf(Rval{
			Url: "http://" + addr + "/reply",
		})
	}
}

func Test_10(t *testing.T) {
	// client trace to log whether the request's underlying tcp connection was re-used
	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			log.Printf("conn was reused: %t", info.Reused)
		},
	}
	traceCtx := httptrace.WithClientTrace(context.Background(), clientTrace)

	reuse.Reqf(Rval{
		Ctx: traceCtx,
		Url: "https://www.baidu.com/1",
	})
	reuse.Reqf(Rval{
		Ctx: traceCtx,
		Url: "https://www.bilibili.com/",
	})
	reuse.Reqf(Rval{
		Ctx: traceCtx,
		Url: "https://www.baidu.com/3",
	})
	reuse.Reqf(Rval{
		Ctx: traceCtx,
		Url: "https://www.bilibili.com/",
	})
}

func Test_9(t *testing.T) {
	reuse.Reqf(Rval{
		Url:                   "http://" + addr + "1/nores",
		ResponseHeaderTimeout: 500,
	})
	if reuse.ResStatusCode() != 0 {
		t.Fatal()
	}
}

func Test_7(t *testing.T) {
	e := reuse.Reqf(Rval{
		Url:                   "http://" + addr + "/nores",
		ResponseHeaderTimeout: 500,
	})
	if !IsTimeout(e) {
		t.Fatal(e)
	}
}

func Test_6(t *testing.T) {
	if e := reuse.Reqf(Rval{
		Url: "http://" + addr + "/header",
		Header: map[string]string{
			`I`: `1`,
		},
	}); e != nil {
		t.Fatal(e)
	}
	reuse.Response(func(r *http.Response) error {
		if r.Header.Get(`I`) != `1` {
			t.Fail()
		}
		return nil
	})
}

func Test_8(t *testing.T) {
	reuse.Reqf(Rval{
		Url:     "http://" + addr + "/reply",
		PostStr: "123",
	})
	reuse.Respon(func(b []byte) error {
		if !bytes.Equal([]byte("123"), b) {
			t.Fatal()
		}
		return nil
	})
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/reply",
	})
	reuse.Respon(func(b []byte) error {
		if bytes.Equal([]byte("123"), b) {
			t.Fatal()
		}
		return nil
	})
}

// go test -timeout 30s -run ^Test_reuse$ github.com/qydysky/part/reqf -race -count=1 -v -memprofile mem.out
func Test_reuse(t *testing.T) {
	for i := 0; i < 20; i++ {
		reuse.Reqf(Rval{
			Url: "http://" + addr + "/no",
		})
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
		reuse.Respon(func(buf []byte) error {
			if !bytes.Equal([]byte("abc强强强强"), buf) {
				b.Fail()
			}
			return nil
		})
	}
}

func Test15(t *testing.T) {
	i, o := io.Pipe()

	if e := reuse.Reqf(Rval{
		Url:                 "http://" + addr + "/stream",
		NoResponse:          true,
		SaveToPipe:          pio.NewPipeRaw(i, o),
		Async:               true,
		CopyResponseTimeout: 100,
	}); e != nil {
		t.Log(e)
	}

	buf := make([]byte, 1<<8)
	for {
		if n, e := i.Read(buf); n != 0 {
			continue
		} else if e != nil {
			break
		}
	}

	if !errors.Is(reuse.Wait(), ErrCopyRes) {
		t.Fatal()
	}
}

func Test14(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	i, o := io.Pipe()

	r := New()
	if e := r.Reqf(Rval{
		Url:                 "http://" + addr + "/stream",
		Ctx:                 ctx,
		NoResponse:          true,
		SaveToPipe:          pio.NewPipeRaw(i, o),
		Async:               true,
		CopyResponseTimeout: 5*1000*2 + 1,
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
	ctx, ctxc := context.WithCancel(context.Background())
	r := New()
	r.Reqf(Rval{
		Ctx:   ctx,
		Url:   "http://" + addr + "/to",
		Async: true,
	})
	ctxc()
	if !IsCancel(r.Wait()) {
		t.Error("async Cancel fail")
	}
}

func Benchmark_req10(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// reuse.Reqf(Rval{
		// 	Url: "http://" + addr + "/br",
		// })
		// reuse.Respon(func(buf []byte) error {
		// 	if !bytes.Equal([]byte("abc强强强强"), buf) {
		// 		b.Fail()
		// 	}
		// 	return nil
		// })
		reuse.Reqf(Rval{
			Url: "http://" + addr + "/gzip",
		})
		reuse.Respon(func(buf []byte) error {
			if !bytes.Equal([]byte("abc强强强强"), buf) {
				b.Error("gzip fail")
			}
			return nil
		})
	}
}

func Test_req(t *testing.T) {
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/br",
	})
	reuse.Respon(func(buf []byte) error {
		if !bytes.Equal([]byte("abc强强强强"), buf) {
			t.Fail()
		}
		return nil
	})
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/gzip",
	})
	reuse.Respon(func(buf []byte) error {
		if !bytes.Equal([]byte("abc强强强强"), buf) {
			t.Error("gzip fail")
		}
		return nil
	})
	reuse.Reqf(Rval{
		Url: "http://" + addr + "/flate",
	})
	reuse.Respon(func(buf []byte) error {
		if !bytes.Equal([]byte("abc强强强强"), buf) {
			t.Error("flate fail")
		}
		return nil
	})
}

type J struct {
	A int `json:"a"`
}

func Test_req12(t *testing.T) {
	reuse.Reqf(Rval{
		Url:     "http://" + addr + "/json",
		Timeout: 10 * 1000,
		Retry:   2,
	})
	var j J
	reuse.Respon(func(buf []byte) error {
		if json.Unmarshal(buf, &j) != nil {
			t.Error("json fail")
		}
		return nil
	})
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
	ctx, ctxc := context.WithCancel(context.Background())
	r := New()
	{
		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			ctxc()
		}()
		e := r.Reqf(Rval{
			Ctx: ctx,
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
	ctx, ctxc := context.WithCancel(context.Background())
	r := New()
	{
		rc, wc := io.Pipe()
		go func() {
			var buf []byte = make([]byte, 1<<16)
			_, _ = rc.Read(buf)
			time.Sleep(time.Millisecond * 500)
			ctxc()
		}()
		r.Reqf(Rval{
			Ctx:        ctx,
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
		r.Respon(func(buf []byte) error {
			if !bytes.Equal(buf, []byte("abc强强强强")) {
				t.Error("async fail", buf)
			}
			return nil
		})
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
		r.Respon(func(buf []byte) error {
			if len(buf) != 0 {
				t.Error("io async fail", buf)
			}
			return nil
		})
		wg.Wait()
	}
}

func Test_req5(t *testing.T) {
	r := New()
	r.Reqf(Rval{
		Url:     "http://" + addr + "/reply",
		PostStr: "123",
	})
	r.Respon(func(buf []byte) error {
		if !bytes.Equal(buf, []byte("123")) {
			t.Fatal()
		}
		return nil
	})

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
		t.Fatal()
	}
	reuse.Response(func(r *http.Response) error {
		if _, e := ResDate(r); e != nil {
			t.Fatal()
		}
		return nil
	})
}
