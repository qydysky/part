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
