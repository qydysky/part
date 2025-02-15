package part

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func check(l *RWMutex, r int32) {
	// if i := l.rlc.Load(); i != r {
	// 	panic(fmt.Errorf("%v %v", i, r))
	// }
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
	t.Parallel()
	var l RWMutex
	//check(&l, 0)
	ul := l.RLock()
	//check(&l, 2)
	ul1 := l.RLock()
	//check(&l, 3)
	ul()
	//check(&l, 2)
	ul1()
	//check(&l, 0)
}

func Test4(t *testing.T) {
	t.Parallel()
	var l RWMutex
	ul := l.RLock()
	ul(func(ulocked bool) (doUlock bool) {
		ul1 := l.RLock()
		ul1()
		return true
	})
}

func Test5(t *testing.T) {
	t.Parallel()
	var l = RWMutex{PanicFunc: func(a any) {
		if !errors.Is(a.(error), ErrTimeoutToURLock) {
			t.Fatal(a)
		}
	}}
	ul := l.RLock(time.Second, time.Second)
	ul(func(ulocked bool) (doUlock bool) {
		time.Sleep(time.Second * 2)
		return true
	})
}

func Test8(t *testing.T) {
	t.Parallel()
	var l = RWMutex{PanicFunc: func(a any) {
		if !errors.Is(a.(error), ErrTimeoutToRLock) {
			panic(a)
		}
	}}
	ul := l.Lock()
	go ul(func(ulocked bool) (doUlock bool) { time.Sleep(time.Second); return true })
	ul1 := l.RLock(time.Millisecond*500, time.Second)
	ul1()
}

// ulock rlock lock
func Test2(t *testing.T) {
	t.Parallel()
	var l RWMutex
	ul := l.RLock()
	//check(&l, 2)
	time.AfterFunc(time.Second, func() {
		//check(&l, 2)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock()
	//check(&l, -1)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	//check(&l, 0)
}

// ulock lock rlock
func Test3(t *testing.T) {
	t.Parallel()
	var l RWMutex
	ul := l.Lock()
	//check(&l, -1)
	time.AfterFunc(time.Second, func() {
		//check(&l, -1)
		ul()
	})
	c := time.Now()
	ul1 := l.RLock()
	//check(&l, 2)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	//check(&l, 0)
}

func Test6(t *testing.T) {
	t.Parallel()
	var c = make(chan int, 3)
	var l RWMutex
	ul := l.RLock()
	time.AfterFunc(time.Millisecond*500, func() {
		ul1 := l.RLock()
		c <- 1
		ul1()
		c <- 2
	})
	ul(func(ulocked bool) (doUlock bool) {
		time.Sleep(time.Second)
		return true
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
	t.Parallel()
	var c = make(chan int, 2)
	var l RWMutex
	ul := l.RLock()
	ul1 := l.RLock()
	ul(func(ulocked bool) (doUlock bool) {
		c <- 0
		return true
	})
	ul1(func(ulocked bool) (doUlock bool) {
		c <- 1
		return true
	})
	if <-c != 0 {
		t.Fatal()
	}
	if <-c != 1 {
		t.Fatal()
	}
}

func Test9(t *testing.T) {
	t.Parallel()
	n := time.Now()
	var l RWMutex
	for i := 0; i < 1000; i++ {
		go l.RLock(time.Second, time.Second)()
	}
	t.Log(time.Since(n))
}

func Test10(t *testing.T) {
	t.Parallel()
	n := time.Now()
	var l sync.RWMutex
	for i := 0; i < 30000; i++ {
		l.RLock()
		go l.RUnlock()
	}
	t.Log(time.Since(n))

	n = time.Now()
	var l2 RWMutex
	for i := 0; i < 30000; i++ {
		go l2.RLock()()
	}
	t.Log(time.Since(n))
}

func Test11(t *testing.T) {
	t.Parallel()
	var l RWMutex

	l.RLock()(func(ulocked bool) (doUlock bool) {
		return true
	}, func(ulocked bool) (doUlock bool) {
		if ulocked {
			defer l.Lock()()
		}
		return
	})
	l.RLock()(func(ulocked bool) (doUlock bool) {
		return false
	}, func(ulocked bool) (doUlock bool) {
		if ulocked {
			defer l.Lock()()
		}
		return
	})
}

func Panic_Test8(t *testing.T) {
	t.Parallel()
	var l RWMutex
	ul := l.Lock(time.Second, time.Second)
	ul(func(ulocked bool) (doUlock bool) {
		time.Sleep(time.Second * 10)
		return true
	})
}

// ulock rlock rlock
func Test4Panic_(t *testing.T) {
	t.Parallel()
	var l RWMutex
	l.RecLock(10, time.Second, time.Second)
	//check(&l, 0)
	ul := l.RLock(time.Second, time.Second)
	//check(&l, 1)
	ul1 := l.RLock(time.Second, time.Second)
	//check(&l, 1)
	ul2 := l.RLock(time.Second, time.Second)
	//check(&l, 2)
	time.Sleep(time.Millisecond * 1500)
	ul()
	//check(&l, 1)
	ul1()
	ul2()
	//check(&l, 0)
	time.Sleep(time.Second * 3)
}

// ulock rlock lock
func Test5Panic_(t *testing.T) {
	t.Parallel()
	var l RWMutex
	l.RecLock(10, time.Second, time.Second)

	l.RLock()()
	ul := l.RLock()
	//check(&l, 1)
	time.AfterFunc(time.Millisecond*1500, func() {
		//check(&l, 1)
		ul()
	})
	c := time.Now()
	ul1 := l.Lock(time.Second * 2)
	//check(&l, 0)
	if time.Since(c) < time.Second {
		t.Fail()
	}
	ul1()
	//check(&l, 0)
}

/*
goos: linux
goarch: amd64
pkg: github.com/qydysky/part/sync
cpu: Intel(R) Celeron(R) J4125 CPU @ 2.00GHz
BenchmarkRlock
BenchmarkRlock-4

	1000000              1069 ns/op              24 B/op          1 allocs/op

# PASS

goos: linux
goarch: amd64
pkg: github.com/qydysky/part/sync
cpu: Intel(R) N100
BenchmarkRlock
BenchmarkRlock-4

	4493745               268.9 ns/op            48 B/op          1 allocs/op

PASS
*/
func BenchmarkRlock(b *testing.B) {
	var lock1 RWMutex
	lock1.RecLock(5, time.Second, time.Second)
	for i := 0; i < b.N; i++ {
		lock1.RLock()()
	}
}

/*
goos: linux
goarch: amd64
pkg: github.com/qydysky/part/sync
cpu: Intel(R) N100
BenchmarkRlock1
BenchmarkRlock1-4

	7057746               158.2 ns/op             0 B/op          0 allocs/op

PASS
*/
func BenchmarkRlock1(b *testing.B) {
	var lock1 sync.RWMutex
	for i := 0; i < b.N; i++ {
		lock1.RLock()
		_ = 1
		lock1.RUnlock()
	}
}
