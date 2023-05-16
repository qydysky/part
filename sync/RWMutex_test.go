package part

import (
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	var rl RWMutex
	var callL time.Time
	var callRL time.Time
	var callRL2 time.Time
	var to = time.Second * 2

	ul := rl.RLock(to)()
	callRL = time.Now()

	rlock := rl.RLock(to)
	go func() {
		unlock := rlock()
		callRL2 = time.Now()
		unlock()
	}()

	lock := rl.Lock(to)
	go func() {
		ull := lock()
		callL = time.Now()
		ull()
	}()

	time.Sleep(time.Second)
	ul()
	rl.Lock(to)()()

	if time.Since(callRL) < time.Since(callRL2) {
		t.Fatal()
	}
	if time.Since(callRL2) < time.Since(callL) {
		t.Fatal()
	}
	if callL.IsZero() {
		t.Fatal()
	}
}

func BenchmarkRlock(b *testing.B) {
	var lock1 RWMutex
	for i := 0; i < b.N; i++ {
		lock1.Lock(time.Second)()()
	}
}
