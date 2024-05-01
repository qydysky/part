package part

import (
	"sync"
	"sync/atomic"
	"time"
)

type Map struct {
	size atomic.Int64
	m    sync.Map
}

func (t *Map) Store(k, v any) {
	if _, loaded := t.m.Swap(k, v); !loaded {
		t.size.Add(1)
	}
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

func (t *Map) Delete(k any) (ok bool) {
	if _, ok := t.m.LoadAndDelete(k); ok {
		t.size.Add(-1)
		return true
	}
	return false
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

type MapExceeded[K, V any] struct {
	m Map
}

type mapExceededItem[V any] struct {
	data     *V
	exceeded time.Time
	wait     sync.RWMutex
}

func (t *MapExceeded[K, V]) Store(k K, v *V, dur time.Duration) {
	t.m.Store(k, &mapExceededItem[V]{
		data:     v,
		exceeded: time.Now().Add(dur),
	})
}

func (t *MapExceeded[K, V]) Load(k K) (*V, bool) {
	if v, ok := t.m.LoadV(k).(*mapExceededItem[V]); ok {
		if v.exceeded.After(time.Now()) {
			return v.data, true
		}
		t.Delete(k)
	}
	return nil, false
}

func (t *MapExceeded[K, V]) Range(f func(key K, value *V) bool) {
	t.m.Range(func(key, value any) bool {
		if value.(*mapExceededItem[V]).exceeded.After(time.Now()) {
			return f(key.(K), value.(*V))
		}
		t.Delete(key.(K))
		return true
	})
}

func (t *MapExceeded[K, V]) Len() int {
	return t.m.Len()
}

func (t *MapExceeded[K, V]) GC() {
	t.m.Range(func(key, value any) bool {
		if value.(*mapExceededItem[V]).exceeded.Before(time.Now()) {
			t.Delete(key.(K))
		}
		return true
	})
}

func (t *MapExceeded[K, V]) Delete(k K) {
	t.m.Delete(k)
}

func (t *MapExceeded[K, V]) LoadOrStore(k K) (vr *V, loaded bool, store func(v1 *V, dur time.Duration)) {
	store = func(v1 *V, dur time.Duration) {}
	var actual any
	actual, loaded = t.m.LoadOrStore(k, &mapExceededItem[V]{})
	v := actual.(*mapExceededItem[V])
	v.wait.RLock()
	exp := v.exceeded
	vr = v.data
	v.wait.RUnlock()
	if loaded && time.Now().Before(exp) {
		return
	}
	if !loaded || (loaded && !exp.IsZero()) {
		store = func(v1 *V, dur time.Duration) {
			v.wait.Lock()
			v.data = v1
			v.exceeded = time.Now().Add(dur)
			v.wait.Unlock()
		}
		return
	}
	for loaded && exp.IsZero() {
		time.Sleep(time.Millisecond * 20)
		v.wait.RLock()
		exp = v.exceeded
		vr = v.data
		v.wait.RUnlock()
	}
	return
}
