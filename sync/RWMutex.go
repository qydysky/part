package part

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	lock  = -1
	ulock = 0
)

type RWMutex struct {
	rlc atomic.Int32
	cul atomic.Int32
	oll atomic.Int32
}

func (m *RWMutex) RLock(to ...time.Duration) (unrlock func()) {
	if len(to) > 0 {
		c := time.Now()
		for m.rlc.Load() < ulock {
			runtime.Gosched()
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait rlock, rlc:%d", m.rlc.Load()))
			}
		}
	} else {
		for m.rlc.Load() < ulock {
			runtime.Gosched()
			time.Sleep(time.Millisecond)
		}
	}
	m.rlc.Add(1)
	var callC atomic.Bool
	return func() {
		if !callC.CompareAndSwap(false, true) {
			panic("had unrlock")
		}
		m.rlc.Add(-1)
	}
}

func (m *RWMutex) Lock(to ...time.Duration) (unlock func()) {
	lockid := m.cul.Add(1)

	if len(to) > 0 {
		c := time.Now()
		if !m.rlc.CompareAndSwap(ulock, lock) {
			for m.rlc.Load() > ulock {
				runtime.Gosched()
				if time.Since(c) > to[0] {
					panic(fmt.Sprintf("timeout to wait rlock, rlc:%d", m.rlc.Load()))
				}
			}
			for lockid-1 != m.oll.Load() {
				runtime.Gosched()
				if time.Since(c) > to[0] {
					panic(fmt.Sprintf("timeout to wait lock, rlc:%d", m.rlc.Load()))
				}
			}
			if !m.rlc.CompareAndSwap(ulock, lock) {
				panic("csa error, bug")
			}
		}
	} else {
		if !m.rlc.CompareAndSwap(ulock, lock) {
			for m.rlc.Load() > ulock {
				runtime.Gosched()
				time.Sleep(time.Millisecond)
			}
			for lockid-1 != m.oll.Load() {
				runtime.Gosched()
				time.Sleep(time.Millisecond)
			}
			if !m.rlc.CompareAndSwap(ulock, lock) {
				panic("")
			}
		}
	}
	var callC atomic.Bool
	return func() {
		if !callC.CompareAndSwap(false, true) {
			panic("had unlock")
		}
		if !m.rlc.CompareAndSwap(lock, ulock) {
			panic("")
		}
		m.oll.Store(lockid)
	}
}
