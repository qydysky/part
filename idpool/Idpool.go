package part

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Idpool struct {
	pool sync.Pool
	sum  int64
}

type Id struct {
	Id   uintptr
	Item interface{}
}

func New(f func() interface{}) *Idpool {
	return &Idpool{
		pool: sync.Pool{
			New: func() interface{} {
				var o = new(Id)
				o.Item = f()
				o.Id = uintptr(unsafe.Pointer(&o.Item))
				return o
			},
		},
	}
}

func (t *Idpool) Get() (o *Id) {
	o = t.pool.Get().(*Id)
	atomic.AddInt64(&t.sum, 1)
	return
}

func (t *Idpool) Put(i *Id) {
	if i.Item == nil {
		return
	}
	i.Item = nil
	t.pool.Put(i)
	atomic.AddInt64(&t.sum, -1)
}

func (t *Idpool) Len() int64 {
	return atomic.LoadInt64(&t.sum)
}
