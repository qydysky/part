package part

import (
	"testing"
	p "github.com/qydysky/part"
)

func Test_msgq(t *testing.T) {
	mq := New()
	go func(){
		for mq.Pull(0).(string) == `mmm` {;}
		t.Error(`0`)
	}()
	go func(){
		for {
			o := mq.Pull(1)
			if o.(string)  == `mmm` || o == nil {continue}
			break
		}
		t.Error(`1`)
	}()
	p.Sys().Timeoutf(1)
	mq.Push(`mmm`)
	p.Sys().Timeoutf(1)
	mq.Cancle(1)
	mq.Push(`mmm`)
}
