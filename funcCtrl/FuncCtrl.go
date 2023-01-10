package part

import (
	"container/list"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	idpool "github.com/qydysky/part/idpool"
)

type SkipFunc struct { //新的跳过
	c unsafe.Pointer
}

func (t *SkipFunc) NeedSkip() (result bool) {
	return !atomic.CompareAndSwapPointer(&t.c, nil, unsafe.Pointer(&struct{}{}))
}

func (t *SkipFunc) UnSet() {
	atomic.CompareAndSwapPointer(&t.c, atomic.LoadPointer(&t.c), nil)
}

type FlashFunc struct { //新的替换旧的
	b    *list.List
	pool *idpool.Idpool
}

func (t *FlashFunc) Flash() (current uintptr) {
	if t.pool == nil {
		t.pool = idpool.New(func() interface{} { return new(struct{}) })
	}
	if t.b == nil {
		t.b = list.New()
	}

	e := t.pool.Get()
	current = e.Id
	t.b.PushFront(e)

	return
}

func (t *FlashFunc) UnFlash() {
	t.b.Remove(t.b.Back())
}

func (t *FlashFunc) NeedExit(current uintptr) bool {
	return current != t.b.Front().Value.(*idpool.Id).Id
}

type BlockFunc struct { //新的等待旧的
	sync.Mutex
}

func (t *BlockFunc) Block() {
	t.Lock()
}

func (t *BlockFunc) UnBlock() {
	t.Unlock()
}

type BlockFuncN struct { //新的等待旧的 个数
	plan int64
	n    int64
	Max  int64
}

func (t *BlockFuncN) Block() {
	for {
		now := atomic.LoadInt64(&t.n)
		if now < t.Max && now >= 0 {
			break
		}
		runtime.Gosched()
	}
	atomic.AddInt64(&t.n, 1)
}

func (t *BlockFuncN) UnBlock() {
	for {
		now := atomic.LoadInt64(&t.n)
		if now > 0 {
			break
		}
		runtime.Gosched()
	}
	atomic.AddInt64(&t.n, -1)
	if atomic.LoadInt64(&t.plan) > 0 {
		atomic.AddInt64(&t.plan, -1)
	}
}

func (t *BlockFuncN) None() {
	for !atomic.CompareAndSwapInt64(&t.n, 0, -1) {
		runtime.Gosched()
	}
}

func (t *BlockFuncN) UnNone() {
	for !atomic.CompareAndSwapInt64(&t.n, -1, 0) {
		runtime.Gosched()
	}
}

func (t *BlockFuncN) Plan(n int64) {
	for !atomic.CompareAndSwapInt64(&t.plan, 0, n) {
		runtime.Gosched()
	}
}

func (t *BlockFuncN) PlanDone(switchFuncs ...func()) {
	for atomic.LoadInt64(&t.plan) > 0 {
		for i := 0; i < len(switchFuncs); i++ {
			switchFuncs[i]()
		}
		runtime.Gosched()
	}
}
