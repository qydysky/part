package part

import (
	"sync"
)

type Buf[T any] struct {
	maxsize int
	newF    func() *T
	inUse   func(*T) bool
	reuseF  func(*T) *T
	poolF   func(*T) *T
	buf     []*T
	l       sync.RWMutex
}

// 创建池
//
// NewF: func() *T 新值
//
// InUse func(*T) bool 是否在使用
//
// ReuseF func(*T) *T 重用前处理
//
// PoolF func(*T) *T 入池前处理
//
// maxsize int 池最大数量
func New[T any](NewF func() *T, InUse func(*T) bool, ReuseF func(*T) *T, PoolF func(*T) *T, maxsize int) *Buf[T] {
	t := new(Buf[T])
	t.newF = NewF
	t.inUse = InUse
	t.reuseF = ReuseF
	t.poolF = PoolF
	t.maxsize = maxsize
	return t
}

func (t *Buf[T]) PoolInUse() (inUse int) {
	t.l.RLock()
	defer t.l.RUnlock()

	for i := 0; i < len(t.buf); i++ {
		if t.inUse(t.buf[i]) {
			inUse++
		}
	}

	return
}

func (t *Buf[T]) PoolSum() int {
	t.l.RLock()
	defer t.l.RUnlock()

	return len(t.buf)
}

func (t *Buf[T]) Trim() {
	t.l.Lock()
	defer t.l.Unlock()

	for i := 0; i < len(t.buf); i++ {
		if !t.inUse(t.buf[i]) {
			t.buf[i] = nil
			t.buf = append(t.buf[:i], t.buf[i+1:]...)
			i--
		}
	}
}

func (t *Buf[T]) Get() *T {
	t.l.Lock()
	defer t.l.Unlock()

	for i := 0; i < len(t.buf); i++ {
		if !t.inUse(t.buf[i]) {
			return t.reuseF(t.buf[i])
		}
	}

	return t.newF()
}

func (t *Buf[T]) Put(item ...*T) {
	if len(item) == 0 {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	var cu = 0
	for i := 0; i < len(t.buf); i++ {
		if !t.inUse(t.buf[i]) {
			t.buf[i] = t.poolF(item[cu])
			cu++
			if cu >= len(item) {
				return
			}
		}
	}

	for i := cu; i < len(item) && t.maxsize > len(t.buf); i++ {
		t.buf = append(t.buf, t.poolF(item[i]))
	}
}
