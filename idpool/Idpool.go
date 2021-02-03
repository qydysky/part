package part

import (
	"sync"
	"unsafe"
)

type Idpool struct {
	pool sync.Pool
	sum uint
	sync.Mutex
}

type Id struct {
	Id uintptr
	item interface{}
}

func New() (*Idpool) {
	return &Idpool{
		pool:sync.Pool{
			New: func() interface{} {
				return new(struct{})
			},
		},
	}
}

func (t *Idpool) Get() (o *Id) {
	o = new(Id)
	o.item = t.pool.Get()
	o.Id = uintptr(unsafe.Pointer(&o.item))
	t.Lock()
	t.sum += 1
	t.Unlock()
	return
}

func (t *Idpool) Put(i *Id) {
	if i.item == nil {return}
	t.pool.Put(i.item)
	i.item = nil
	t.Lock()
	t.sum -= 1
	t.Unlock()
}

func (t *Idpool) Len() uint {
	return t.sum
}