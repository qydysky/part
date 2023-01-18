package part

import (
	"sync"
)

type buf[T any] struct {
	maxsize int
	newF    func() *T
	validF  func(*T) bool
	reuseF  func(*T) *T
	buf     []*T
	sync.RWMutex
}

func New[T any](NewF func() *T, ValidF func(*T) bool, ReuseF func(*T) *T, maxsize int) *buf[T] {
	t := new(buf[T])
	t.newF = NewF
	t.validF = ValidF
	t.reuseF = ReuseF
	t.maxsize = maxsize
	return t
}

func (t *buf[T]) Trim() {
	t.Lock()
	defer t.Unlock()

	for i := 0; i < len(t.buf); i++ {
		if !t.validF(t.buf[i]) {
			t.buf[i] = nil
			t.buf = append(t.buf[:i], t.buf[i:]...)
			i--
		}
	}
}

func (t *buf[T]) Get() *T {
	t.Lock()
	defer t.Unlock()

	for i := 0; i < len(t.buf); i++ {
		if !t.validF(t.buf[i]) {
			return t.reuseF(t.buf[i])
		}
	}

	return t.newF()
}

func (t *buf[T]) Put(item ...*T) {
	if len(item) == 0 {
		return
	}

	t.Lock()
	defer t.Unlock()

	var cu = 0
	for i := 0; i < len(t.buf); i++ {
		if !t.validF(t.buf[i]) {
			t.buf[i] = item[cu]
			cu++
			if cu >= len(item) {
				return
			}
		}
	}

	for i := cu; i < len(item) && t.maxsize > len(t.buf); i++ {
		t.buf = append(t.buf, item[i])
	}
}
