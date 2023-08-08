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
	mbuf    map[*T]bool
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
	t.mbuf = make(map[*T]bool)
	return t
}

// states[] 0:pooled, 1:nopooled, 2:inuse, 3:nouse, 4:sum
func (t *Buf[T]) PoolState() (states []any) {
	t.l.RLock()
	defer t.l.RUnlock()

	var pooled, nopooled, inuse, nouse, sum int

	sum = len(t.mbuf)
	for k, v := range t.mbuf {
		if v {
			pooled++
		} else {
			nopooled++
		}
		if t.inUse(k) {
			inuse++
		} else {
			nouse++
		}
	}

	return []any{pooled, nopooled, inuse, nouse, sum}
}

func (t *Buf[T]) Get() *T {
	t.l.Lock()
	defer t.l.Unlock()

	for k, v := range t.mbuf {
		if v && !t.inUse(k) {
			t.mbuf[k] = true
			return t.reuseF(k)
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

	for i := 0; i < len(item); i++ {
		if _, ok := t.mbuf[item[i]]; ok {
			t.poolF(item[i])
			t.mbuf[item[i]] = true
		} else if t.maxsize > len(t.mbuf) {
			t.poolF(item[i])
			t.mbuf[item[i]] = true
		}
	}
}
