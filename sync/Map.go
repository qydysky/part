package part

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Map struct {
	size atomic.Int64
	m    sync.Map
}

func (t *Map) Store(k, v any) {
	t.size.Add(1)
	t.m.Store(k, v)
}

func (t *Map) Load(k any) (any, bool) {
	return t.m.Load(k)
}

func (t *Map) LoadV(k any) (v any) {
	v, _ = t.m.Load(k)
	return
}

func (t *Map) Range(f func(key, value any) bool) {
	t.m.Range(f)
}

func (t *Map) Delete(k any) {
	if _, ok := t.m.LoadAndDelete(k); ok {
		t.size.Add(-1)
	}
}

func (t *Map) Len() int {
	return int(t.size.Load())
}

type ptr struct {
	p unsafe.Pointer
}

func (t *ptr) tryStore(v *any) {
	t.p = unsafe.Pointer(v)
	// atomic.StorePointer(&t.p, unsafe.Pointer(v))
}

func (t *ptr) tryLoad() (any, bool) {
	// p := atomic.LoadPointer(&t.p)
	if t.p == nil {
		return nil, false
	}
	return *(*any)(t.p), true
}

type pLock struct {
	i    unsafe.Pointer
	busy unsafe.Pointer
}

func (l *pLock) Lock() {
	if l.busy == nil {
		l.busy = unsafe.Pointer(&struct{}{})
	}
	for !atomic.CompareAndSwapPointer(&l.i, nil, l.busy) {
		runtime.Gosched()
	}
}

func (l *pLock) Locking() bool {
	return atomic.LoadPointer(&l.i) != nil
}

func (l *pLock) Unlock() {
	atomic.CompareAndSwapPointer(&l.i, l.busy, nil)
}
