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

func Test_Timeout(t *testing.T) {
	r := New()
	if e := r.Reqf(Rval{
		Url:     `https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
		Timeout: 1000,
	}); e != nil {
		if !IsTimeout(e) {
			t.Error(`type error`, e)
		}
		return
	}
	t.Log(`no error`)
}

func Test_Cancel(t *testing.T) {
	r := New()

	go func() {
		time.Sleep(time.Second)
		r.Cancel()
	}()

	if e := r.Reqf(Rval{
		Url: `https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
	}); e != nil {
		if !IsCancel(e) {
			t.Error(`type error`, e)
		}
		return
	}
	t.Log(`no error`)
}

func Test_Cancel_chan(t *testing.T) {
	r := New()

	c := make(chan []byte, 1<<16)

	go func() {
		for {
			<-c
		}
	}()

	go func() {
		time.Sleep(time.Second * 3)
		r.Cancel()
	}()

	if e := r.Reqf(Rval{
		Url:        `https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
		SaveToChan: c,
		Timeout:    5000,
	}); e != nil {
		if !IsCancel(e) {
			t.Error(`type error`, e)
		}
		return
	}
	t.Log(`no error`)
}

func Test_Io_Pipe(t *testing.T) {
	r := New()
	rp, wp := io.Pipe()
	c := make(chan struct{}, 1)
	go func() {
		buf, _ := io.ReadAll(rp)
		t.Log("Test_Io_Pipe download:", len(buf))
		t.Log("Test_Io_Pipe download:", len(r.Respon))
		close(c)
	}()
	if e := r.Reqf(Rval{
		Url:              `https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
		SaveToPipeWriter: wp,
		Timeout:          5000,
	}); e != nil {
		if !IsTimeout(e) {
			t.Error(`type error`, e)
		}
		return
	}
	t.Log(`no error`)
	<-c
}

func Test_compress(t *testing.T) {
	addr := "127.0.0.1:10001"
	s := web.New(&http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * time.Duration(10),
	})
	s.Handle(map[string]func(http.ResponseWriter, *http.Request){
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
}
