package part

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Idpool[T any] struct {
	pool  sync.Pool
	sum   int64
	initF func() *T
}

type Id[T any] struct {
	Id   uintptr
	Item *T
}

func New[T any](f func() *T) *Idpool[T] {
	return &Idpool[T]{
		initF: f,
		pool: sync.Pool{
			New: func() any {
				var o = new(Id[T])
				o.Item = f()
				o.Id = uintptr(unsafe.Pointer(&o.Item))
				return o
			},
		},
	}
}

func (t *Idpool[T]) Get() (o *Id[T]) {
	o = t.pool.Get().(*Id[T])
	if o.Item == nil {
		o.Item = t.initF()
		o.Id = uintptr(unsafe.Pointer(&o.Item))
	}
	atomic.AddInt64(&t.sum, 1)
	return
}

func (t *Idpool[T]) Put(is ...*Id[T]) {
	for _, i := range is {
		if i.Item == nil {
			continue
		}
		t.pool.Put(i)
		atomic.AddInt64(&t.sum, -1)
	}
}

func (t *Idpool[T]) InUse() int64 {
	return atomic.LoadInt64(&t.sum)
}
