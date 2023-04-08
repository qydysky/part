package part

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// 新的跳过
type SkipFunc struct {
	c unsafe.Pointer
}

func (t *SkipFunc) NeedSkip() (result bool) {
	return !atomic.CompareAndSwapPointer(&t.c, nil, unsafe.Pointer(&struct{}{}))
}

func (t *SkipFunc) UnSet() {
	atomic.CompareAndSwapPointer(&t.c, atomic.LoadPointer(&t.c), nil)
}

// 新的替换旧的
type FlashFunc struct {
	b atomic.Uintptr
	c *context.CancelFunc
}

func (t *FlashFunc) Flash() (current uintptr) {
	current = uintptr(unsafe.Pointer(&struct{}{}))
	t.b.Store(current)
	if t.c != nil {
		(*t.c)()
		t.c = nil
	}
	return
}

func (t *FlashFunc) UnFlash() {}

func (t *FlashFunc) NeedExit(current uintptr) bool {
	return t.b.Load() != current
}

func (t *FlashFunc) FlashWithContext() (current uintptr, c context.Context) {
	current = uintptr(unsafe.Pointer(&struct{}{}))
	t.b.Store(current)
	if t.c != nil {
		(*t.c)()
		t.c = nil
	}
	c, cancle := context.WithCancel(context.Background())
	t.c = &cancle
	return
}

// 新的等待旧的
type BlockFunc struct {
	sync.Mutex
}

func (t *BlockFunc) Block() {
	t.Lock()
}

func (t *BlockFunc) UnBlock() {
	t.Unlock()
}

type BlockFuncN struct { //新的等待旧的 个数
	n   atomic.Int64
	Max int64
}

func (t *BlockFuncN) Block(failF ...func()) {
	for {
		now := t.n.Load()
		if now < t.Max && now >= 0 {
			break
		}
		for i := 0; i < len(failF); i++ {
			failF[i]()
		}
		runtime.Gosched()
	}
	t.n.Add(1)
}

func (t *BlockFuncN) UnBlock(failF ...func()) {
	for {
		now := t.n.Load()
		if now > 0 {
			break
		}
		for i := 0; i < len(failF); i++ {
			failF[i]()
		}
		runtime.Gosched()
	}
	t.n.Add(-1)
}

func (t *BlockFuncN) BlockAll(failF ...func()) {
	for !t.n.CompareAndSwap(0, -1) {
		for i := 0; i < len(failF); i++ {
			failF[i]()
		}
		runtime.Gosched()
	}
}

func (t *BlockFuncN) UnBlockAll() {
	if !t.n.CompareAndSwap(-1, 0) {
		panic("must BlockAll First")
	}
}
