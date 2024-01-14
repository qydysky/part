package part

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// const (
// 	lock  int32 = -1
// 	ulock int32 = 0
// 	rlock int32 = 1
// )

var (
	ErrTimeoutToLock  = errors.New("ErrTimeoutToLock")
	ErrTimeoutToULock = errors.New("ErrTimeoutToULock")
)

type RWMutex struct {
	rlc       sync.RWMutex
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
	callTree := getCall(2)
	return time.AfterFunc(to, func() {
		m.panicFunc(errors.Join(e, errors.New(callTree)))
	}).Stop
}

// to[0]: wait lock timeout
//
// to[1]: wait ulock timeout
//
// 不要在Rlock内设置变量，有DATA RACE风险
func (m *RWMutex) RLock(to ...time.Duration) (unlockf func(beforeUlock ...func())) {
	if len(to) > 0 {
		defer m.tof(to[0], ErrTimeoutToLock)()
	}

	m.rlc.RLock()

	return func(beforeUlock ...func()) {
		if len(to) > 1 {
			defer m.tof(to[1], ErrTimeoutToULock)()
		}
		for i := 0; i < len(beforeUlock); i++ {
			beforeUlock[i]()
		}
		m.rlc.RUnlock()
	}
}

// to[0]: wait lock timeout
//
// to[1]: wait ulock timeout
func (m *RWMutex) Lock(to ...time.Duration) (unlockf func(beforeUlock ...func())) {
	if len(to) > 0 {
		defer m.tof(to[0], ErrTimeoutToLock)()
	}

	m.rlc.Lock()

	return func(beforeUlock ...func()) {
		if len(to) > 1 {
			defer m.tof(to[1], ErrTimeoutToULock)()
		}
		for i := 0; i < len(beforeUlock); i++ {
			beforeUlock[i]()
		}
		m.rlc.Unlock()
	}
}

func getCall(i int) (calls string) {
	for i += 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, runtime.GOROOT()) {
			break
		} else {
			calls += fmt.Sprintf("\ncall by %s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	return
}
