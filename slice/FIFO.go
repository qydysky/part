package part

import (
	"sync"

	"errors"
)

var (
	ErrFIFOOverflow = errors.New(`ErrFIFOOverflow`)
	ErrFIFOEmpty    = errors.New(`ErrFIFOEmpty`)
)

type FIFOSize interface {
	int | int32 | int64 | uint | uint32 | uint64
}

type item struct {
	op, ed int
}

type FIFO[S any] struct {
	ed, op, opc int
	c           chan item
	buf         []S
	l           sync.RWMutex
}

func NewFIFO[S any, T FIFOSize](size T) *FIFO[S] {
	return &FIFO[S]{
		c:   make(chan item, size),
		buf: make([]S, size),
	}
}

func (t *FIFO[S]) lock() func() {
	t.l.Lock()
	return t.l.Unlock
}

func (t *FIFO[S]) rlock() func() {
	t.l.RLock()
	return t.l.RUnlock
}

func (t *FIFO[S]) inok(size int) bool {
	if t.ed+size > len(t.buf) {
		if size > t.op {
			return false
		}
		t.ed = 0
	} else if t.op > t.ed && t.ed+size > t.op {
		return false
	}
	return true
}

func (t *FIFO[S]) In(p []S) error {
	defer t.lock()()

	t.op = t.opc
	if !t.inok(len(p)) {
		return ErrFIFOOverflow
	}
	select {
	case t.c <- item{
		op: t.ed,
		ed: t.ed + len(p),
	}:
		t.ed = t.ed + copy(t.buf[t.ed:], p)
	default:
		return ErrFIFOOverflow
	}
	return nil
}

func (t *FIFO[S]) Out(w interface {
	Write(p []S) (n int, err error)
}) (n int, err error) {
	defer t.rlock()()

	select {
	case item := <-t.c:
		n, err = w.Write(t.buf[item.op:item.ed])
		t.opc = item.ed
	default:
		err = ErrFIFOEmpty
	}

	return
}

func (t *FIFO[S]) OutDirect() (p []S, err error, used func()) {
	used = t.rlock()

	select {
	case item := <-t.c:
		p = t.buf[item.op:item.ed]
		t.opc = item.ed
	default:
		err = ErrFIFOEmpty
	}

	return
}

func (t *FIFO[S]) Size() int {
	defer t.rlock()()

	if t.opc > t.ed {
		return len(t.buf) - t.opc - t.ed
	} else {
		return t.ed - t.opc
	}
}

func (t *FIFO[S]) Num() int {
	return len(t.c)
}

func (t *FIFO[S]) Clear() {
	defer t.lock()()

	t.op = 0
	t.opc = 0
	t.ed = 0
	for {
		select {
		case <-t.c:
		default:
			return
		}
	}
}

func (t *FIFO[S]) Reset() {
	defer t.lock()()

	clear(t.buf)
	t.op = 0
	t.opc = 0
	t.ed = 0
	for {
		select {
		case <-t.c:
		default:
			return
		}
	}
}
