package part

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	compress "github.com/qydysky/part/compress"
	web "github.com/qydysky/part/web"
)

func Test_req(t *testing.T) {
	addr := "127.0.0.1:10001"
	s := web.New(&http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * time.Duration(10),
	})
	s.Handle(map[string]func(http.ResponseWriter, *http.Request){
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
		`/exit`: func(_ http.ResponseWriter, _ *http.Request) {
			s.Server.Shutdown(context.Background())
		},
	})

	r := New()
	r.Reqf(Rval{
		Url: "http://" + addr + "/br",
	})
	if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
		t.Error("br fail")
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
	{
		e := r.Reqf(Rval{
			Url:     "http://" + addr + "/to",
			Timeout: 1000,
		})
		if !IsTimeout(e) {
			t.Error("Timeout fail")
		}
	}
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
			Async:            true,
		})
		<-c
	}
	{
		r.Reqf(Rval{
			Url:   "http://" + addr + "/flate",
			Async: true,
		})
		if len(r.Respon) != 0 {
			t.Error("async fail")
		}
		r.Wait()
		if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
			t.Error("async fail")
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
		d, _ := io.ReadAll(rc)
		if !bytes.Equal(d, []byte("abc强强强强")) {
			t.Error("io async fail")
		}
		if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
			t.Error("io async fail")
		}
	}
}
