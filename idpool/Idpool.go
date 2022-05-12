package part

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Idpool struct {
	pool  sync.Pool
	sum   int64
	initF func() interface{}
}

type Id struct {
	Id   uintptr
	Item interface{}
}

func New(f func() interface{}) *Idpool {
	return &Idpool{
		initF: f,
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
	if o.Item == nil {
		o.Item = t.initF()
		o.Id = uintptr(unsafe.Pointer(&o.Item))
	}
	atomic.AddInt64(&t.sum, 1)
	return
}

func (t *Idpool) Put(i *Id) {
	if i.Item == nil {
		return
	}
	t.pool.Put(i)
	atomic.AddInt64(&t.sum, -1)
}

func (t *Idpool) Len() int64 {
	return atomic.LoadInt64(&t.sum)
}
