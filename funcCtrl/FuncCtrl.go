package part

import (
	"sync"
	"unsafe"
	"sync/atomic"
	"container/list"
	idpool "github.com/qydysky/part/idpool"
)

type SkipFunc struct{//新的跳过
	c unsafe.Pointer
}

func (t *SkipFunc) NeedSkip() (result bool) {
	return !atomic.CompareAndSwapPointer(&t.c, nil, unsafe.Pointer(&struct{}{}))
}

func (t *SkipFunc) UnSet() {
	atomic.CompareAndSwapPointer(&t.c, atomic.LoadPointer(&t.c), nil)
}

type FlashFunc struct{//新的替换旧的
	b *list.List
	pool *idpool.Idpool
}

func (t *FlashFunc) Flash() (current uintptr) {
	if t.pool == nil {t.pool = idpool.New()}
	if t.b == nil {t.b = list.New()}

	e := t.pool.Get()
	current = e.Id
	t.b.PushFront(e)
	
	return
}

func (t *FlashFunc) NeedExit(current uintptr) (bool) {
	return current != t.b.Front().Value.(*idpool.Id).Id
}

type BlockFunc struct{//新的等待旧的
	sync.Mutex
}

func (t *BlockFunc) Block() {
	t.Lock()
}

func (t *BlockFunc) UnBlock() {
	t.Unlock()
}