package part

import (
	"testing"
	p "github.com/qydysky/part"
)

func Test_msgq(t *testing.T) {
	mq := New()
	k := 0
	var e bool
	for i:=0;i<1e5;i++{
		go func(){
			k += 1
			if o,ok:=mq.Pull().(string);o != `mmm`||!ok {e = true}
			k += 1
		}()
	}
	p.Sys().Timeoutf(2)
	t.Log(`>`,k)
	k = 0

	mq.Push(`mmm`)

	p.Sys().Timeoutf(1)
	t.Log(`<`,k)
	if e {t.Error("f")}
}
