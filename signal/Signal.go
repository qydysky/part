package part

import (
	"runtime"
	"sync/atomic"
)

type Signal struct {
	c         chan struct{}
	waitCount atomic.Int32
}

func Init() *Signal {
	return &Signal{c: make(chan struct{})}
}

func (i *Signal) Wait() {
	if i.Islive() {
		i.waitCount.Add(1)
		<-i.c
		i.waitCount.Add(-1)
	}
}

// unsafe. fin() need
func (i *Signal) WaitC() (c chan struct{}, fin func()) {
	if i.Islive() {
		i.waitCount.Add(1)
		return i.c, i.fin
	}
	return nil, func() {}
}

func (i *Signal) fin() {
	i.waitCount.Add(-1)
}

func (i *Signal) Done() {
	if i.Islive() {
		close(i.c)
		for !i.waitCount.CompareAndSwap(0, -1) {
			runtime.Gosched()
		}
	}
}

func (i *Signal) Islive() (islive bool) {
	if i == nil {
		return
	}
	select {
	case <-i.c: //close
	default: //still alive
		if i.c == nil {
			break
		} //not make yet
		islive = true //has made
	}
	return
}
