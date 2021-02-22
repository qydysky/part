package part

import (
	"testing"
	"time"
)

func Test_SkipFunc(t *testing.T) {
	var b SkipFunc
	var a = func(i int){
		if b.NeedSkip() {return}
		defer b.UnSet()
		t.Log(i,`.`)
		time.Sleep(time.Second)
		t.Log(i,`..`)
	}
	t.Log(`just show 1 or 2 twice`)
	go a(1)
	go a(2)
	time.Sleep(5*time.Second)
}

func Test_FlashFunc(t *testing.T) {
	var b FlashFunc
	var a = func(i int){
		id := b.Flash()
		t.Log(i,`.`)
		time.Sleep(time.Second)
		if b.NeedExit(id) {return}
		t.Log(i,`.`)
	}
	t.Log(`show 1 or 2, then show the other twice`)
	go a(1)
	go a(2)
	time.Sleep(5*time.Second)
}

func Test_BlockFunc(t *testing.T) {
	var b BlockFunc
	var a = func(i int){
		b.Block()
		defer b.UnBlock()
		t.Log(i,`.`)
		time.Sleep(time.Second)
		t.Log(i,`.`)
	}
	t.Log(`show 1 and 2 twice`)
	go a(1)
	go a(2)
	time.Sleep(5*time.Second)
}