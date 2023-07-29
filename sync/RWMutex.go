package part

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

const (
	lock  int32 = -1
	ulock int32 = 0
	rlock int32 = 1
)

type RWMutex struct {
	rlc  atomic.Int32
	read atomic.Int32
}

func parse(i int32) string {
	switch i {
	case -1:
		return "lock"
	case 0:
		return "ulock"
	case 1:
		return "rlock"
	}
	return "unknow"
}

// i == oldt -> i = t -> pass
//
// otherwish block until i == oldt
func cas(i *atomic.Int32, oldt, t int32) (ok bool, loop func(to ...time.Duration) error) {
	if i.CompareAndSwap(oldt, t) {
		return true, func(to ...time.Duration) error { return nil }
	} else {
		var called atomic.Bool
		return false, func(to ...time.Duration) error {
			if !called.CompareAndSwap(false, true) {
				panic("had called")
			}
			c := time.Now()
			for !i.CompareAndSwap(oldt, t) {
				if len(to) != 0 && time.Since(c) > to[0] {
					return fmt.Errorf("timeout to set %s => %s while is %s", parse(oldt), parse(t), parse(i.Load()))
				}
				runtime.Gosched()
			}
			return nil
		}
	}
}

// i == t -> pass
//
// i == oldt -> i = t -> pass
//
// otherwish block until i == oldt
func lcas(i *atomic.Int32, oldt, t int32) (ok bool, loop func(to ...time.Duration) error) {
	if i.Load() == t || i.CompareAndSwap(oldt, t) {
		return true, func(to ...time.Duration) error { return nil }
	} else {
		var called atomic.Bool
		return false, func(to ...time.Duration) error {
			if !called.CompareAndSwap(false, true) {
				panic("had called")
			}
			c := time.Now()
			for !i.CompareAndSwap(oldt, t) {
				if len(to) != 0 && time.Since(c) > to[0] {
					return fmt.Errorf("timeout to set %s => %s while is %s", parse(oldt), parse(t), parse(i.Load()))
				}
				runtime.Gosched()
			}
			return nil
		}
	}
}

// call inTimeCall() in time or panic(callTree)
func tof(to time.Duration) (inTimeCall func() (called bool)) {
	callTree := getCall(2)
	return time.AfterFunc(to, func() {
		panic("Locking timeout!\n" + callTree)
	}).Stop
}

// to[0]: wait lock timeout to[1]: run lock timeout
//
// 不要在Rlock内设置变量，有DATA RACE风险
func (m *RWMutex) RLock(to ...time.Duration) (unlockf func(beforeUlock ...func())) {
	if m.read.Add(1) == 1 {
		_, rlcLoop := cas(&m.rlc, ulock, rlock)
		if e := rlcLoop(to...); e != nil {
			panic(e)
		}
	} else {
		_, rlcLoop := lcas(&m.rlc, ulock, rlock)
		if e := rlcLoop(to...); e != nil {
			panic(e)
		}
	}
	var callC atomic.Bool
	var done func() (called bool)
	if len(to) > 1 {
		done = tof(to[1])
	}
	return func(beforeUlock ...func()) {
		if !callC.CompareAndSwap(false, true) {
			panic("had unlock")
		}
		if done != nil {
			done()
		}
		if m.read.Add(-1) == 0 {
			for i := 0; i < len(beforeUlock); i++ {
				beforeUlock[i]()
			}
			_, rlcLoop := cas(&m.rlc, rlock, ulock)
			if e := rlcLoop(to...); e != nil {
				panic(e)
			}
		}
	}
}

// to[0]: wait lock timeout to[1]: run lock timeout
func (m *RWMutex) Lock(to ...time.Duration) (unlockf func(beforeUlock ...func())) {
	_, rlcLoop := cas(&m.rlc, ulock, lock)
	if e := rlcLoop(to...); e != nil {
		panic(e)
	}
	var callC atomic.Bool
	var done func() (called bool)
	if len(to) > 1 {
		done = tof(to[1])
	}
	return func(beforeUlock ...func()) {
		if !callC.CompareAndSwap(false, true) {
			panic("had unlock")
		}
		if done != nil {
			done()
		}
		for i := 0; i < len(beforeUlock); i++ {
			beforeUlock[i]()
		}
		_, rlcLoop := cas(&m.rlc, lock, ulock)
		if e := rlcLoop(to...); e != nil {
			panic(e)
		}
	}
}

func getCall(i int) (calls string) {
	for i += 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, runtime.GOROOT()) {
			break
		} else {
			calls += fmt.Sprintf("call by %s\n\t%s:%d\n", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	return
}
