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

	ul := rl.RLock(to)
	callRL = time.Now()

	go func() {
		ull := rl.RLock(to)
		callRL2 = time.Now()
		ull()
	}()

	go func() {
		ull := rl.Lock(to)
		callL = time.Now()
		ull()
	}()

	time.Sleep(time.Second)
	ul()
	rl.Lock(to)()

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
