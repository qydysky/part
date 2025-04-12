package part

import (
	"sync"
	"time"
)

type MapExceeded[K, V any] struct {
	m Map
}

type mapExceededItem[V any] struct {
	data     V
	exceeded time.Time
	wait     sync.RWMutex
}

func (t *MapExceeded[K, V]) Copy() (m *MapExceeded[K, V]) {
	m = &MapExceeded[K, V]{}
	t.m.Range(func(key, value any) bool {
		if value.(*mapExceededItem[V]).exceeded.After(time.Now()) {
			m.m.Store(key, value)
		}
		return true
	})
	return
}

func (t *MapExceeded[K, V]) Store(k K, v V, dur time.Duration) {
	t.m.Store(k, &mapExceededItem[V]{
		data:     v,
		exceeded: time.Now().Add(dur),
	})
}

func (t *MapExceeded[K, V]) Load(k K) (v V, ok bool) {
	if v, ok := t.m.LoadV(k).(*mapExceededItem[V]); ok {
		if v.exceeded.After(time.Now()) {
			return v.data, true
		}
		t.Delete(k)
	}
	return
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

func (t *MapExceeded[K, V]) LoadOrStore(k K) (vr V, loaded bool, store func(v1 V, dur time.Duration)) {
	store = func(v1 V, dur time.Duration) {}
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
		store = func(v1 V, dur time.Duration) {
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
