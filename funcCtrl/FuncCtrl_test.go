package part

import (
	"testing"
	"time"
)

func Test_SkipFunc(t *testing.T) {
	var c = make(chan int, 2)
	var b SkipFunc
	var a = func(i int) {
		if b.NeedSkip() {
			return
		}
		defer b.UnSet()
		c <- i
		time.Sleep(time.Second)
		c <- i
	}
	go a(1)
	go a(2)
	if i0 := <-c; i0 != <-c {
		t.Fatal()
	}
}

func Test_FlashFunc(t *testing.T) {
	var c = make(chan int, 2)
	var b FlashFunc
	var a = func(i int) {
		id := b.Flash()
		defer b.UnFlash()
		c <- i
		time.Sleep(time.Second)
		if b.NeedExit(id) {
			return
		}
		c <- i
	}
	go a(1)
	go a(2)
	if i0 := <-c; i0 == <-c {
		t.Fatal()
	}
}

func Test_BlockFunc(t *testing.T) {
	var c = make(chan int, 2)
	var b BlockFunc
	var a = func(i int) {
		b.Block()
		defer b.UnBlock()
		c <- i
		time.Sleep(time.Second)
		c <- i
	}
	go a(1)
	go a(2)
	if i0 := <-c; i0 != <-c {
		t.Fatal()
	}
	if i0 := <-c; i0 != <-c {
		t.Fatal()
	}
}

func Test_BlockFuncN(t *testing.T) {
	var c = make(chan string, 8)
	var cc string

	var b = &BlockFuncN{
		Max: 2,
	}
	var a = func(i string) {
		b.Block()
		defer b.UnBlock()
		c <- i
		time.Sleep(time.Second)
		c <- i
	}
	go a("0")
	time.Sleep(time.Millisecond * 20)
	go a("1")
	time.Sleep(time.Millisecond * 20)
	go a("2")

	for len(c) > 0 {
		cc += <-c
	}
	if cc != "01" {
		t.Fatal()
	}

	b.BlockAll()
	b.UnBlockAll()

	for len(c) > 0 {
		cc += <-c
	}
	if cc != "010212" {
		t.Fatal()
	}
	// t.Log(cc)
}
