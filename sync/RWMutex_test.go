package part

import (
	"testing"
	"time"
)

func check(l *RWMutex, r, read int32) {
	if l.rlc.Load() != r {
		panic("rlc")
	}
	if l.read.Load() != read {
		panic("read")
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
	check(&l, ulock, 0)
	ul := l.RLock()
	check(&l, rlock, 1)
	ul1 := l.RLock()
	check(&l, rlock, 2)
	ul()
	check(&l, rlock, 1)
	ul1()
	check(&l, ulock, 0)
}

// ulock rlock lock
func Test2(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	check(&l, rlock, 1)
	time.AfterFunc(time.Second, func() {
		check(&l, rlock, 1)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock()
	check(&l, lock, 0)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, ulock, 0)
}

// ulock lock rlock
func Test3(t *testing.T) {
	var l RWMutex
	ul := l.Lock()
	check(&l, lock, 0)
	time.AfterFunc(time.Second, func() {
		check(&l, lock, 1)
		ul()
	})
	c := time.Now()
	ul1 := l.RLock()
	check(&l, rlock, 1)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, ulock, 0)
}

func Test6(t *testing.T) {
	var c = make(chan int, 2)
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
	if <-c != 0 {
		t.Fatal()
	}
	if <-c != 1 {
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
	if <-c != 1 {
		t.Fatal()
	}
}

// ulock rlock rlock
func Panic_Test4(t *testing.T) {
	var l RWMutex
	check(&l, ulock, 0)
	ul := l.RLock(time.Second, time.Second)
	check(&l, rlock, 1)
	ul1 := l.RLock(time.Second, time.Second)
	check(&l, rlock, 2)
	time.Sleep(time.Millisecond * 1500)
	ul()
	check(&l, rlock, 1)
	ul1()
	check(&l, ulock, 0)
	time.Sleep(time.Second * 3)
}

// ulock rlock lock
func Panic_Test5(t *testing.T) {
	var l RWMutex
	ul := l.RLock()
	check(&l, rlock, 1)
	time.AfterFunc(time.Millisecond*1500, func() {
		check(&l, rlock, 1)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock(time.Second)
	check(&l, lock, 0)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	check(&l, ulock, 0)
}

func BenchmarkRlock(b *testing.B) {
	var lock1 RWMutex
	var a bool
	for i := 0; i < b.N; i++ {
		ul := lock1.RLock()
		a = true
		ul()
	}
	println(a)
}
