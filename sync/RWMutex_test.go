package part

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func check(l *RWMutex, r int32) {
	if i := l.rlc.Load(); i != r {
		panic(fmt.Errorf("%v %v", i, r))
	}
}

func Test0(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	go func() {
		l.Lock()()
	}()
	go func() {
		l.RLock()()
	}()
	ul()
}

// ulock rlock rlock
func Test1(t *testing.T) {
	var l RWMutex
	check(&l, 0)
	ul := l.RLock()
	check(&l, 2)
	ul1 := l.RLock()
	check(&l, 3)
	ul()
	check(&l, 2)
	ul1()
	check(&l, 0)
}

func Test4(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	ul(func() {
		ul1 := l.RLock()
		ul1()
	})
}

func Test5(t *testing.T) {
	var l = RWMutex{PanicFunc: func(a any) {
		if !errors.Is(a.(error), ErrTimeoutToULock) {
			t.Fatal(a)
		}
	}}
	ul := l.RLock(time.Second, time.Second)
	ul(func() {
		time.Sleep(time.Second * 2)
	})
}

func Test8(t *testing.T) {
	var l = RWMutex{PanicFunc: func(a any) {
		if !errors.Is(a.(error), ErrTimeoutToLock) {
			panic(a)
		}
	}}
	ul := l.Lock()
	go ul(func() { time.Sleep(time.Second) })
	ul1 := l.RLock(time.Millisecond*500, time.Second)
	ul1()
}

// ulock rlock lock
func Test2(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	check(&l, 2)
	time.AfterFunc(time.Second, func() {
		check(&l, 2)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock()
	check(&l, -1)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, 0)
}

// ulock lock rlock
func Test3(t *testing.T) {
	var l RWMutex
	ul := l.Lock()
	check(&l, -1)
	time.AfterFunc(time.Second, func() {
		check(&l, -1)
		ul()
	})
	c := time.Now()
	ul1 := l.RLock()
	check(&l, 2)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, 0)
}

func Test6(t *testing.T) {
	var c = make(chan int, 3)
	var l RWMutex
	ul := l.RLock()
	time.AfterFunc(time.Millisecond*500, func() {
		ul1 := l.RLock()
		c <- 1
		ul1()
		c <- 2
	})
	ul(func() {
		time.Sleep(time.Second)
	})
	c <- 0
	if <-c != 1 {
		t.Fatal()
	}
	if <-c != 2 {
		t.Fatal()
	}
	if <-c != 0 {
		t.Fatal()
	}
}

func Test7(t *testing.T) {
	var c = make(chan int, 2)
	var l RWMutex
	ul := l.RLock()
	ul1 := l.RLock()
	ul(func() {
		c <- 0
	})
	ul1(func() {
		c <- 1
	})
	if <-c != 0 {
		t.Fatal()
	}
	if <-c != 1 {
		t.Fatal()
	}
}

func Panic_Test8(t *testing.T) {
	var l RWMutex
	ul := l.Lock(time.Second, time.Second)
	ul(func() {
		time.Sleep(time.Second * 10)
	})
}

// ulock rlock rlock
func Panic_Test4(t *testing.T) {
	var l RWMutex
	check(&l, 0)
	ul := l.RLock(time.Second, time.Second)
	check(&l, 1)
	ul1 := l.RLock(time.Second, time.Second)
	check(&l, 2)
	time.Sleep(time.Millisecond * 1500)
	ul()
	check(&l, 1)
	ul1()
	check(&l, 0)
	time.Sleep(time.Second * 3)
}

// ulock rlock lock
func Panic_Test5(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	check(&l, 1)
	time.AfterFunc(time.Millisecond*1500, func() {
		check(&l, 1)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock(time.Second)
	check(&l, 0)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, 0)
}

/*
goos: linux
goarch: amd64
pkg: github.com/qydysky/part/sync
cpu: Intel(R) Celeron(R) J4125 CPU @ 2.00GHz
BenchmarkRlock
BenchmarkRlock-4

	1000000              1069 ns/op              24 B/op          1 allocs/op

PASS
*/
func BenchmarkRlock(b *testing.B) {
	var lock1 RWMutex
	for i := 0; i < b.N; i++ {
		lock1.Lock()()
	}
}
