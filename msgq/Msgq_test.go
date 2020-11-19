package part

import (
	"testing"
	p "github.com/qydysky/part"
)

func Test_msgq(t *testing.T) {


	mq := New(25)
	mun := 1000000
	mun_c := make(chan bool,mun)
	mun_s := make(chan bool,mun)

	var e int

	sig := mq.Sig()
	for i:=0;i<mun;i++{
		go func(){
			mun_c <- true
			data,t0 := mq.Pull(sig)
			if o,ok:=data.(string);o != `mmm`||!ok {e = 1}
			data1,_ := mq.Pull(t0)
			if o,ok:=data1.(string);o != `mm1`||!ok {e = 2}
			mun_s <- true
		}()
	}

	for len(mun_c) != mun {
		t.Log(`>`,len(mun_c))
		p.Sys().Timeoutf(1)
	}
	t.Log(`>`,len(mun_c))

	t.Log(`push mmm`)
	mq.Push(`mmm`)
	t.Log(`push mm1`)
	mq.Push(`mm1`)

	for len(mun_s) != mun {
		t.Log(`<`,len(mun_s))
		p.Sys().Timeoutf(1)
	}
	t.Log(`<`,len(mun_s))

	if e!=0 {t.Error(e)}
}
