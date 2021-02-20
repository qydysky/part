package part

import (
	"unsafe"
	"runtime"
	"sync/atomic"
)

type Map struct {
	lock pLock //sync
	num int
	m map[interface{}]*ptr
	readOnly atomic.Value //sync
}

func (t *Map) Store(k,v interface{}) {

	m,_ := t.readOnly.Load().(map[interface{}]*ptr)
	if p,ok := m[k];ok{
		p.tryStore(&v)
		return
	}

	t.lock.Lock()

	m,_ = t.readOnly.Load().(map[interface{}]*ptr)
	if p,ok := m[k];ok{
		p.tryStore(&v)
	} else if p,ok := t.m[k];ok{
		p.tryStore(&v)
	} else {
		if t.m == nil {t.mapFrom(&m)}
		t.m[k] = &ptr{p:unsafe.Pointer(&v)}
		t.num += 1
	}

	t.lock.Unlock()
}

func (t *Map) Load(k interface{}) (interface{},bool) {

	m,_ := t.readOnly.Load().(map[interface{}]*ptr)
	p,ok := m[k]
	if !ok && t.num != 0{
		t.lock.Lock()

		m,_ = t.readOnly.Load().(map[interface{}]*ptr)
		p,ok = m[k]
		if !ok && t.num != 0{
			if t.m == nil {t.mapFrom(&m)}
			p,ok = t.m[k]
			if ok && t.num > 1e3{
				t.readOnly.Store(t.m)
				t.m = nil
				t.num = 0
			}
		}
	
		t.lock.Unlock()
	}

	if !ok{
		return nil,false
	}

	return p.tryLoad()
}

func (t *Map) LoadV(k interface{}) (v interface{}) {
	v,_ = t.Load(k)
	return
}

func (t *Map) Range(f func(key, value interface{})(bool)) {
	t.lock.Lock()

	m,_ := t.readOnly.Load().(map[interface{}]*ptr)
	if t.m == nil {t.mapFrom(&m)}
	t.readOnly.Store(t.m)
	t.m = nil
	t.num = 0

	t.lock.Unlock()

	m,_ = t.readOnly.Load().(map[interface{}]*ptr)//reload
	for k,p := range m{
		v,ok := p.tryLoad()
		if !ok {continue} 
		if !f(k,v) {return}
	}

	return
}

func (t *Map) Delete(k interface{}) {
	m,_ := t.readOnly.Load().(map[interface{}]*ptr)
	
	if p,ok := m[k];ok && p != nil{
		delete(m, k)
		return
	}

	t.lock.Lock()

	delete(t.m, k)

	t.lock.Unlock()
}

func (t *Map) Len() int {
	m,_ := t.readOnly.Load().(map[interface{}]*ptr)
	return len(m) + t.num
}

func (t *Map) mapFrom(from *map[interface{}]*ptr) {
	if t.m == nil {t.m = make(map[interface{}]*ptr)}
	for k,v := range *from{
		if v == nil {continue}
		t.m[k] = v
	}
}

type ptr struct {
	p unsafe.Pointer
}

func (t *ptr) tryStore(v *interface{}) {
	t.p = unsafe.Pointer(v)
	// atomic.StorePointer(&t.p, unsafe.Pointer(v))
}

func (t *ptr) tryLoad() (interface{},bool) {
	// p := atomic.LoadPointer(&t.p)
	if t.p == nil{
		return nil,false
	}
	return *(*interface{})(t.p),true
}

type pLock struct{
	i unsafe.Pointer
	busy unsafe.Pointer
}

func (l *pLock) Lock() {
	if l.busy == nil{l.busy = unsafe.Pointer(&struct{}{})}
	for !atomic.CompareAndSwapPointer(&l.i, nil, l.busy) {
		runtime.Gosched()
	}
}

func (l *pLock) Locking() (bool) {
	return atomic.LoadPointer(&l.i) != nil
}

func (l *pLock) Unlock() {
	atomic.CompareAndSwapPointer(&l.i, l.busy, nil)
}
