package part

import (
	"errors"
	"fmt"
	"go/build"
	"runtime"
	"strings"
	"sync"
	"time"
	"weak"
)

// const (
// 	lock  int32 = -1
// 	ulock int32 = 0
// 	rlock int32 = 1
// )

var (
	ErrTimeoutToLock   = errors.New("ErrTimeoutToLock")
	ErrTimeoutToULock  = errors.New("ErrTimeoutToULock")
	ErrTimeoutToRLock  = errors.New("ErrTimeoutToRLock")
	ErrTimeoutToURLock = errors.New("ErrTimeoutToURLock")
)

type RWMutex struct {
	rlc       sync.RWMutex
	rlccl     sync.Mutex
	rlcc      []weak.Pointer[string]
	to        []time.Duration
	PanicFunc func(any)
}

// func parse(i int32) string {
// 	switch i {
// 	case -2:
// 		return "lock"
// 	case -1:
// 		return "ulock"
// 	}
// 	return "rlock"
// }

// // i == oldt -> i = t -> pass
// //
// // otherwish block until i == oldt
// func cas(i *atomic.Int32, oldt, t int32, to ...time.Duration) error {
// 	c := time.Now()
// 	for !i.CompareAndSwap(oldt, t) {
// 		if len(to) != 0 && time.Since(c) > to[0] {
// 			return fmt.Errorf("timeout to set %s => %s while is %s", parse(oldt), parse(t), parse(i.Load()))
// 		}
// 		runtime.Gosched()
// 	}
// 	return nil
// }

// // i == t -> pass
// //
// // i == oldt -> i = t -> pass
// //
// // otherwish block until i == oldt
// func lcas(i *atomic.Int32, oldt, t int32, to ...time.Duration) error {
// 	c := time.Now()
// 	for i.Load() != t && !i.CompareAndSwap(oldt, t) {
// 		if len(to) != 0 && time.Since(c) > to[0] {
// 			return fmt.Errorf("timeout to set %s => %s while is %s", parse(oldt), parse(t), parse(i.Load()))
// 		}
// 		runtime.Gosched()
// 	}
// 	return nil
// }

func (m *RWMutex) panicFunc(s any) {
	if m.PanicFunc != nil {
		m.PanicFunc(s)
	} else {
		panic(s)
	}
}

// call inTimeCall() in time or panic(callTree)
func (m *RWMutex) tof(to time.Duration, e error) (inTimeCall func() (called bool)) {
	callTree := getCall(1)
	return time.AfterFunc(to, func() {
		runtime.GC()
		e = errors.Join(e, errors.New(*callTree))
		m.rlccl.Lock()
		for i := 0; i < len(m.rlcc); i++ {
			if s := m.rlcc[i].Value(); s != nil {
				e = errors.Join(e, errors.New("\nlocking:"+*s))
			}
		}
		m.rlccl.Unlock()
		m.panicFunc(e)
	}).Stop
}

func (m *RWMutex) RecLock(max int, to ...time.Duration) {
	defer m.Lock()()
	m.rlcc = make([]weak.Pointer[string], max)
	m.to = to
}

func (m *RWMutex) addcl(s *string) *string {
	m.rlccl.Lock()
	m.rlcc[copy(m.rlcc, m.rlcc[1:])] = weak.Make(s)
	m.rlccl.Unlock()
	return s
}

// to[0]: wait lock timeout
//
// to[1]: wait ulock timeout
//
// 不要在Rlock内设置变量，有DATA RACE风险
func (m *RWMutex) RLock(to ...time.Duration) (unlockf func(ulockfs ...func(ulocked bool) (doUlock bool))) {
	to = append(to, m.to...)
	if len(to) > 0 {
		defer m.tof(to[0], ErrTimeoutToRLock)()
	}

	m.rlc.RLock()
	var ct *string
	if m.rlcc != nil {
		ct = m.addcl(getCall(0))
	}

	return func(ulockfs ...func(ulocked bool) (doUlock bool)) {
		inTimeCall := func() (called bool) { return true }
		if len(to) > 1 {
			inTimeCall = m.tof(to[1], ErrTimeoutToURLock)
		}
		ul := false
		for i := 0; i < len(ulockfs); i++ {
			if ulockfs[i](ul) && !ul {
				m.rlc.RUnlock()
				inTimeCall()
				ul = true
			}
		}
		if !ul {
			m.rlc.RUnlock()
			inTimeCall()
		}
		_ = ct
	}
}

// to[0]: wait lock timeout
//
// to[1]: wait ulock timeout
func (m *RWMutex) Lock(to ...time.Duration) (unlockf func(ulockfs ...func(ulocked bool) (doUlock bool))) {
	to = append(to, m.to...)
	if len(to) > 0 {
		defer m.tof(to[0], ErrTimeoutToLock)()
	}

	m.rlc.Lock()
	var ct *string
	if m.rlcc != nil {
		ct = m.addcl(getCall(0))
	}

	return func(ulockfs ...func(ulocked bool) (doUlock bool)) {
		inTimeCall := func() (called bool) { return true }
		if len(to) > 1 {
			inTimeCall = m.tof(to[1], ErrTimeoutToULock)
		}
		ul := false
		for i := 0; i < len(ulockfs); i++ {
			if ulockfs[i](ul) && !ul {
				m.rlc.Unlock()
				inTimeCall()
				ul = true
			}
		}
		if !ul {
			m.rlc.Unlock()
			inTimeCall()
		}
		_ = ct
	}
}

func getCall(i int) (calls *string) {
	var cs string
	for i += 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, build.Default.GOROOT) {
			break
		} else {
			cs += fmt.Sprintf("\ncall by %s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	return &cs
}
