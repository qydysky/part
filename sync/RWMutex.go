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
	var callC atomic.Bool
	if len(to) > 0 {
		var calls []string
		for i := 1; true; i++ {
			if pc, file, line, ok := runtime.Caller(i); !ok {
				break
			} else {
				calls = append(calls, fmt.Sprintf("%s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line))
			}
		}
		c := time.Now()
		for m.rlc.Load() < ulock || m.cul.Load() != m.oll.Load() {
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait lock, rlc:%d", m.rlc.Load()))
			}
			runtime.Gosched()
		}
		c = time.Now()
		go func() {
			for !callC.Load() {
				if time.Since(c) > to[0] {
					panicS := fmt.Sprintf("timeout to run rlock %v > %v\n", time.Since(c), to[0])
					for i := 0; i < len(calls); i++ {
						panicS += fmt.Sprintf("%s\n", calls[i])
					}
					panic(panicS)
				}
				runtime.Gosched()
			}
		}()
	} else {
		for m.rlc.Load() < ulock || m.cul.Load() != m.oll.Load() {
			time.Sleep(time.Millisecond)
			runtime.Gosched()
		}
	}
	m.rlc.Add(1)
	return func() {
		if !callC.CompareAndSwap(false, true) {
			panic("had unrlock")
		}
		m.rlc.Add(-1)
	}
}

func (m *RWMutex) Lock(to ...time.Duration) (unlock func()) {
	lockid := m.cul.Add(1)
	var callC atomic.Bool
	if len(to) > 0 {
		var calls []string
		for i := 1; true; i++ {
			if pc, file, line, ok := runtime.Caller(i); !ok {
				break
			} else {
				calls = append(calls, fmt.Sprintf("%s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line))
			}
		}
		c := time.Now()
		for m.rlc.Load() > ulock {
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait rlock, rlc:%d", m.rlc.Load()))
			}
			runtime.Gosched()
		}
		for lockid-1 != m.oll.Load() {
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait lock, rlc:%d", m.rlc.Load()))
			}
			runtime.Gosched()
		}
		if !m.rlc.CompareAndSwap(ulock, lock) {
			panic(fmt.Sprintf("csa error, rlc:%d", m.rlc.Load()))
		}
		c = time.Now()
		go func() {
			for !callC.Load() {
				if time.Since(c) > to[0] {
					panicS := fmt.Sprintf("timeout to run lock %v > %v\n", time.Since(c), to[0])
					for i := 0; i < len(calls); i++ {
						panicS += fmt.Sprintf("call by %s\n", calls[i])
					}
					panic(panicS)
				}
				runtime.Gosched()
			}
		}()
	} else {
		for m.rlc.Load() > ulock {
			time.Sleep(time.Millisecond)
			runtime.Gosched()
		}
		for lockid-1 != m.oll.Load() {
			time.Sleep(time.Millisecond)
			runtime.Gosched()
		}
		if !m.rlc.CompareAndSwap(ulock, lock) {
			panic(fmt.Sprintf("csa error, rlc:%d", m.rlc.Load()))
		}
	}
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
