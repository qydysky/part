package part

import (
	"sync"
	"sync/atomic"
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

func Copy[T comparable, S any](s map[T]S) map[T]S {
	t := make(map[T]S)
	for k, v := range s {
		t[k] = v
	}
	return t
}
