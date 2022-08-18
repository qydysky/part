package part

import (
	"io"
	"testing"
	"time"
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
