package part

import (
	"sync"
	"sync/atomic"
)

type MapFunc[T, E any] interface {
	Clear()
	CompareAndDelete(key T, old E) (deleted bool)
	CompareAndSwap(key T, old E, new E) (swapped bool)
	Delete(key T)
	Load(key T) (value E, ok bool)
	LoadAndDelete(key T) (value E, loaded bool)
	LoadOrStore(key T, value E) (actual E, loaded bool)
	Range(f func(key T, value E) bool)
	Store(key T, value E)
	Swap(key T, value E) (previous E, loaded bool)
}

var _ = MapFunc[any, any](&Map{})

type Map struct {
	size atomic.Int64
	m    sync.Map
}

func (t *Map) Store(k, v any) {
	if _, loaded := t.m.Swap(k, v); !loaded {
		t.size.Add(1)
	}
}

func (t *Map) CompareAndSwap(key any, old any, new any) (swapped bool) {
	return t.m.CompareAndSwap(key, old, new)
}

func (t *Map) CompareAndDelete(key any, old any) (deleted bool) {
	deleted = t.m.CompareAndDelete(key, old)
	if deleted {
		t.size.Add(-1)
	}
	return
}

func (t *Map) LoadAndDelete(key any) (value any, loaded bool) {
	value, loaded = t.m.LoadAndDelete(key)
	if loaded {
		t.size.Add(-1)
	}
	return
}

func (t *Map) Swap(key any, value any) (previous any, loaded bool) {
	return t.m.Swap(key, value)
}

func (t *Map) LoadOrStore(k, v any) (actual any, loaded bool) {
	actual, loaded = t.m.LoadOrStore(k, v)
	if !loaded {
		t.size.Add(1)
	}
	return
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

func (t *Map) Clear() {
	t.ClearAll()
}

func (t *Map) ClearAll() {
	t.m.Range(func(key, _ any) bool {
		t.m.Delete(key)
		return true
	})
	t.size.Store(0)
}

func (t *Map) Len() int {
	return int(t.size.Load())
}

func (t *Map) Copy() (m Map) {
	t.Range(func(k, v any) bool {
		m.Store(k, v)
		return true
	})
	return
}

func (t *Map) CopyP() (m *Map) {
	m = &Map{}
	t.Range(func(k, v any) bool {
		m.Store(k, v)
		return true
	})
	return
}

func Copy[T comparable, S any](s map[T]S) map[T]S {
	t := make(map[T]S)
	for k, v := range s {
		t[k] = v
	}
	return t
}

func StoreAll[T comparable, S any, A, B any](d MapFunc[A, B], s map[T]S) {
	for k, v := range s {
		d.Store(any(k).(A), any(v).(B))
	}
}

func Contains[A, B any](m MapFunc[A, B], keys ...A) (missKey []A) {
	for _, tk := range keys {
		if _, ok := m.Load(tk); !ok {
			missKey = append(missKey, tk)
		}
	}
	return
}
