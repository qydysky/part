package part

import (
	"context"
	"iter"
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
	l sync.Mutex
	c *context.CancelFunc
	f *func()
}

func (t *FlashFunc) Flash() (current uintptr) {
	current = uintptr(unsafe.Pointer(&struct{}{}))
	t.b.Store(current)
	return
}

func (t *FlashFunc) UnFlash() {}

func (t *FlashFunc) NeedExit(current uintptr) bool {
	return t.b.Load() != current
}

func (t *FlashFunc) FlashWithContext() (c context.Context) {
	t.l.Lock()
	defer t.l.Unlock()

	if t.c != nil {
		(*t.c)()
		t.c = nil
	}
	c, cancle := context.WithCancel(context.Background())
	t.c = &cancle
	return
}

func (t *FlashFunc) FlashWithCallback(f func()) {
	t.l.Lock()
	defer t.l.Unlock()

	if t.f != nil {
		(*t.f)()
		t.f = nil
	}
	t.f = &f
}

// 新的等待旧的
type BlockFunc struct {
	l sync.Mutex
}

func (t *BlockFunc) Block() {
	t.l.Lock()
}

func (t *BlockFunc) UnBlock() {
	t.l.Unlock()
}

type BlockFuncN struct { //新的等待旧的 个数
	n chan struct{}
}

func NewBlockFuncN(max int) *BlockFuncN {
	return &BlockFuncN{n: make(chan struct{}, max)}
}

func (t *BlockFuncN) Block() (unBlock func()) {
	t.n <- struct{}{}
	return func() {
		<-t.n
	}
}

func (t *BlockFuncN) BlockAll() (unBlock func()) {
	for i := cap(t.n); i > 0; i-- {
		t.n <- struct{}{}
	}
	return func() {
		for i := cap(t.n); i > 0; i-- {
			<-t.n
		}
	}
}

// 在Range时，同时创建返回子上下文ctx
type RangeSource[T any] iter.Seq[T]

func (i RangeSource[T]) RangeCtx(pctx context.Context) iter.Seq2[context.Context, T] {
	return func(yield func(context.Context, T) bool) {
		for o := range i {
			if pctx.Err() != nil {
				return
			}
			ctx, cancle := context.WithCancel(pctx)
			exit := !yield(ctx, o)
			cancle()
			if exit {
				return
			}
		}
	}
}

func (i RangeSource[T]) RangeCtxCancel(pctx context.Context, cancle context.CancelFunc) iter.Seq2[context.Context, T] {
	return func(yield func(context.Context, T) bool) {
		for o := range i {
			if pctx.Err() != nil {
				return
			}
			ctx, cancle := context.WithCancel(pctx)
			exit := !yield(ctx, o)
			cancle()
			if exit {
				return
			}
		}
	}
}
