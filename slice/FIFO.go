package part

import (
	"sync"

	"errors"
)

var (
	ErrFIFOOverflow = errors.New(`ErrFIFOOverflow`)
	ErrFIFOEmpty    = errors.New(`ErrFIFOEmpty`)
)

type item struct {
	op, ed int
}

type FIFO[S any, T []S] struct {
	ed, op, opc int
	c           chan item
	buf         []S
	l           sync.RWMutex
}

func NewFIFO[S any, T []S](size int) *FIFO[S, T] {
	return &FIFO[S, T]{
		c:   make(chan item, size),
		buf: make([]S, size),
	}
}

func (t *FIFO[S, T]) lock() func() {
	t.l.Lock()
	return t.l.Unlock
}

func (t *FIFO[S, T]) rlock() func() {
	t.l.RLock()
	return t.l.RUnlock
}

func (t *FIFO[S, T]) inok(size int) bool {
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

func (t *FIFO[S, T]) In(p T) error {
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

func (t *FIFO[S, T]) Out() (p T, e error) {
	defer t.rlock()()

	select {
	case item := <-t.c:
		p = t.buf[item.op:item.ed]
		t.opc = item.ed
	default:
		e = ErrFIFOEmpty
	}

	return
}

func (t *FIFO[S, T]) Size() int {
	defer t.rlock()()

	if t.opc > t.ed {
		return len(t.buf) - t.opc - t.ed
	} else {
		return t.ed - t.opc
	}
}

func (t *FIFO[S, T]) Num() int {
	return len(t.c)
}

func (t *FIFO[S, T]) Clear() {
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

func (t *FIFO[S, T]) Reset() {
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
