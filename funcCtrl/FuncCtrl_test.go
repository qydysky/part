package part

import (
	"context"
	"testing"
	"time"
)

func Test_RangeCtx(t *testing.T) {
	var rs RangeSource[int] = func(yield func(int) bool) {
		for i := range 10 {
			if !yield(i) {
				return
			}
		}
	}

	for ctx, val := range rs.RangeCtxCancel(context.WithTimeout(context.Background(), time.Millisecond*2500)) {
		select {
		case <-ctx.Done():
			if val < 2 {
				t.Fatal()
			}
		case <-time.After(time.Second):
			if val > 1 {
				t.Fatal()
			}
		}
	}
}

func Test_RangeCtx2(t *testing.T) {
	var i int
	var rs RangeSource[any] = func(yield func(any) bool) {
		for i < 10 {
			if !yield(nil) {
				return
			}
		}
	}

	for ctx := range rs.RangeCtxCancel(context.WithTimeout(context.Background(), time.Millisecond*2500)) {
		i++
		select {
		case <-ctx.Done():
			if i < 2 {
				t.Fatal()
			}
		case <-time.After(time.Second):
			if i > 2 {
				t.Fatal()
			}
		}
	}
}

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

func Test_FlashFunc2(t *testing.T) {
	var cc = make(chan int, 2)
	var b FlashFunc
	var a = func(i int) {
		c := b.FlashWithContext()
		<-c.Done()
		cc <- i
	}
	go a(1)
	go a(2)
	go a(3)
	time.Sleep(time.Second)
	if len(cc) != 2 && <-cc != 1 && <-cc != 2 {
		t.Fatal(len(cc))
	}
}

func Test_FlashFunc3(t *testing.T) {
	var cc = make(chan int, 2)
	var b FlashFunc
	var a = func(i int) {
		b.FlashWithCallback(func() {
			cc <- i
		})
	}
	go a(1)
	go a(2)
	go a(3)
	time.Sleep(time.Second)
	if len(cc) != 2 && <-cc != 1 && <-cc != 2 {
		t.Fatal(len(cc))
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

	var b = NewBlockFuncN(2)
	var a = func(i string) {
		defer b.Block()()
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

	b.BlockAll()()

	for len(c) > 0 {
		cc += <-c
	}
	if cc != "010212" && cc != "010122" {
		t.Fatal(cc)
	}
	// t.Log(cc)
}
