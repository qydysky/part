package part

import (
	"sync"
	"time"
)

type Buf[T any] struct {
	maxsize     int
	pf          PoolFunc[T]
	mbuf        map[*T]bool
	getPerSec   float64
	periodCount float64
	periodTime  time.Time
	l           sync.RWMutex
}

type PoolFunc[T any] struct {
	// func() *T 新值
	New func() *T
	// func(*T) bool 是否在使用
	InUse func(*T) bool
	// func(*T) *T 重用(出池)前处理
	Reuse func(*T) *T
	// func(*T) *T 入池前处理
	Pool func(*T) *T
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
func New[T any](poolFunc PoolFunc[T], maxsize int) *Buf[T] {
	t := new(Buf[T])
	t.pf = poolFunc
	t.maxsize = maxsize
	t.mbuf = make(map[*T]bool)
	t.periodTime = time.Now()
	return t
}

// states[] 0:pooled, 1:nopooled, 2:inuse, 3:nouse, 4:sum, 5:getPerSec
//
// Deprecated: s
func (t *Buf[T]) PoolState() (states []any) {
	state := t.State()
	return []any{state.Pooled, state.Nopooled, state.Inuse, state.Nouse, state.Sum, state.GetPerSec}
}

type BufState struct {
	Pooled, Nopooled, Inuse, Nouse, Sum int
	GetPerSec                           float64
}

func (t *Buf[T]) State() BufState {
	t.l.RLock()
	defer t.l.RUnlock()

	var pooled, nopooled, inuse, nouse, sum int
	var getPerSec float64

	sum = len(t.mbuf)
	for k, v := range t.mbuf {
		if v {
			pooled++
		} else {
			nopooled++
		}
		if t.pf.InUse(k) {
			inuse++
		} else {
			nouse++
		}
	}

	getPerSec = t.periodCount / 10
	if diff := time.Since(t.periodTime).Seconds(); diff > 1 {
		getPerSec += t.getPerSec / diff
	}

	return BufState{pooled, nopooled, inuse, nouse, sum, getPerSec}
}

func (t *Buf[T]) Get() *T {
	t.l.Lock()
	defer t.l.Unlock()

	t.getPerSec += 1
	if diff := time.Since(t.periodTime).Seconds(); diff > 10 {
		t.periodCount = t.getPerSec
		t.getPerSec = 0
		t.periodTime = time.Now()
	}

	for k, v := range t.mbuf {
		if v && !t.pf.InUse(k) {
			t.mbuf[k] = false
			return t.pf.Reuse(k)
		}
	}

	return t.pf.New()
}

func (t *Buf[T]) Put(item ...*T) {
	if len(item) == 0 {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	for i := 0; i < len(item); i++ {
		if _, ok := t.mbuf[item[i]]; ok {
			t.pf.Pool(item[i])
			t.mbuf[item[i]] = true
		} else if t.maxsize > len(t.mbuf) {
			t.pf.Pool(item[i])
			t.mbuf[item[i]] = true
		}
	}
}
