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
	rlock = 0
)

type RWMutex struct {
	rlc  atomic.Int32
	want atomic.Int32
}

// to[0]: wait lock timeout to[1]: run lock timeout
//
// 不要在Rlock内设置变量，有DATA RACE风险
func (m *RWMutex) RLock(to ...time.Duration) (unlockf func()) {
	getWant := m.want.CompareAndSwap(ulock, rlock)
	var callC atomic.Bool
	if len(to) > 0 {
		var calls []string
		if len(to) > 1 {
			for i := 1; true; i++ {
				if pc, file, line, ok := runtime.Caller(i); !ok {
					break
				} else {
					calls = append(calls, fmt.Sprintf("%s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line))
				}
			}
		}
		c := time.Now()
		for m.rlc.Load() < ulock || !getWant && !m.want.CompareAndSwap(ulock, rlock) {
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait lock while rlocking, rlc:%d, want:%d, getWant:%v", m.rlc.Load(), m.want.Load(), getWant))
			}
			runtime.Gosched()
		}
		if len(to) > 1 {
			time.AfterFunc(to[1], func() {
				if !callC.Load() {
					panicS := fmt.Sprintf("timeout to run rlock %v > %v\n", time.Since(c), to[1])
					for i := 0; i < len(calls); i++ {
						panicS += fmt.Sprintf("call by %s\n", calls[i])
					}
					panic(panicS)
				}
			})
		}
	} else {
		for m.rlc.Load() < ulock || !getWant && m.want.CompareAndSwap(ulock, rlock) {
			runtime.Gosched()
		}
	}
	m.rlc.Add(1)
	return func() {
		if !callC.CompareAndSwap(false, true) {
			panic("had unrlock")
		}
		if m.rlc.Add(-1) == ulock {
			m.want.CompareAndSwap(rlock, ulock)
		}
	}
}

// to[0]: wait lock timeout to[1]: run lock timeout
func (m *RWMutex) Lock(to ...time.Duration) (unlockf func()) {
	getWant := m.want.CompareAndSwap(ulock, lock)
	var callC atomic.Bool
	if len(to) > 0 {
		var calls []string
		if len(to) > 1 {
			for i := 1; true; i++ {
				if pc, file, line, ok := runtime.Caller(i); !ok {
					break
				} else {
					calls = append(calls, fmt.Sprintf("%s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line))
				}
			}
		}
		c := time.Now()
		for m.rlc.Load() != ulock || !getWant && !m.want.CompareAndSwap(ulock, lock) {
			if time.Since(c) > to[0] {
				panic(fmt.Sprintf("timeout to wait rlock while locking, rlc:%d, want:%v, getWant:%v", m.rlc.Load(), m.want.Load(), getWant))
			}
			runtime.Gosched()
		}
		if len(to) > 1 {
			time.AfterFunc(to[1], func() {
				if !callC.Load() {
					panicS := fmt.Sprintf("timeout to run lock %v > %v\n", time.Since(c), to[1])
					for i := 0; i < len(calls); i++ {
						panicS += fmt.Sprintf("call by %s\n", calls[i])
					}
					panic(panicS)
				}
			})
		}
	} else {
		for m.rlc.Load() != ulock || !getWant && m.want.CompareAndSwap(ulock, lock) {
			runtime.Gosched()
		}
	}
	m.rlc.Add(-1)
	return func() {
		if !callC.CompareAndSwap(false, true) {
			panic("had unlock")
		}
		if m.rlc.Add(1) == ulock {
			m.want.CompareAndSwap(lock, ulock)
		}
	}
}
