package part

import (
	"sync"
	"sync/atomic"
)

var (
	_ = MapFunc[any, any](&MapG[any, any]{})
)

type MapG[T, E any] struct {
	size atomic.Int64
	m    sync.Map
}

func (t *MapG[T, E]) Store(k T, v E) {
	if _, loaded := t.m.Swap(k, v); !loaded {
		t.size.Add(1)
	}
}

func (t *MapG[T, E]) CompareAndSwap(key T, old E, new E) (swapped bool) {
	return t.m.CompareAndSwap(key, old, new)
}

func (t *MapG[T, E]) CompareAndDelete(key T, old E) (deleted bool) {
	deleted = t.m.CompareAndDelete(key, old)
	if deleted {
		t.size.Add(-1)
	}
	return
}

func (t *MapG[T, E]) LoadAndDelete(key T) (value E, loaded bool) {
	v, l := t.m.LoadAndDelete(key)
	loaded = l
	value = v.(E)
	if l {
		t.size.Add(-1)
	}
	return
}

func (t *MapG[T, E]) Swap(key T, value E) (previous E, loaded bool) {
	v, l := t.m.Swap(key, value)
	loaded = l
	previous = v.(E)
	return
}

func (t *MapG[T, E]) LoadOrStore(key T, value E) (actual E, loaded bool) {
	v, l := t.m.LoadOrStore(key, value)
	loaded = l
	actual = v.(E)
	if !loaded {
		t.size.Add(1)
	}
	return
}

func (t *MapG[T, E]) Load(k T) (E, bool) {
	v, ok := t.m.Load(k)
	if ok {
		return v.(E), true
	}
	return *new(E), false
}

func (t *MapG[T, E]) Range(f func(key T, value E) bool) {
	t.m.Range(func(key, value any) bool {
		return f(key.(T), value.(E))
	})
}

func (t *MapG[T, E]) Delete(k T) {
	if _, ok := t.m.LoadAndDelete(k); ok {
		t.size.Add(-1)
	}
}

func (t *MapG[T, E]) Clear() {
	t.ClearAll()
}

func (t *MapG[T, E]) ClearAll() {
	t.m.Range(func(key, _ any) bool {
		t.m.Delete(key)
		return true
	})
	t.size.Store(0)
}

func (t *MapG[T, E]) Len() int {
	return int(t.size.Load())
}

func (t *MapG[T, E]) Copy() (m MapG[T, E]) {
	t.Range(func(k T, v E) bool {
		m.Store(k, v)
		return true
	})
	return
}

func (t *MapG[T, E]) CopyP() (m *MapG[T, E]) {
	m = &MapG[T, E]{}
	t.Range(func(k T, v E) bool {
		m.Store(k, v)
		return true
	})
	return
}
