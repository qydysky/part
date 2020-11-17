package part

import (
	"testing"
	p "github.com/qydysky/part"
)

func Test_msgq(t *testing.T) {
	mq := New()
	go func(){
		for mq.Pull().(string) == `mmm` {;}
		t.Error(`0`)
	}()	
	go func(){
		for mq.Pull().(string) == `mmm` {;}
		t.Error(`0`)
	}()
	p.Sys().Timeoutf(1)
	mq.Push(`mmm`)
}
