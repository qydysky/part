package part

import (
	"testing"
	"time"
)

func Test_Timeout(t *testing.T) {
	r := Req()
	if e := r.Reqf(Rval{
		Url:`https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
		Timeout:1,
	});e != nil {
		if !IsTimeout(e) {
			t.Error(`type error`,e)
		}
		return
	}
	t.Error(`no error`)
}

func Test_Cancel(t *testing.T) {
	r := Req()

	go func(){
		time.Sleep(time.Second)
		r.Cancel()
	}()

	if e := r.Reqf(Rval{
		Url:`https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.9.0-amd64-netinst.iso`,
	});e != nil {
		if !IsCancel(e) {
			t.Error(`type error`,e)
		}
		return
	}
	t.Error(`no error`)
}