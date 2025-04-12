package part

import "sync"

type LoadOrInitFunc[T any] struct {
	Map interface {
		LoadOrStore(k, v any) (actual any, loaded bool)
	}
	Init  func() *T
	cache *T
	l     sync.Mutex
}

func NewLoadOrInitFunc[T any](m interface {
	LoadOrStore(k, v any) (actual any, loaded bool)
}) *LoadOrInitFunc[T] {
	return &LoadOrInitFunc[T]{
		Map: m,
	}
}

func (l *LoadOrInitFunc[T]) SetInit(init func() *T) *LoadOrInitFunc[T] {
	l.l.Lock()
	defer l.l.Unlock()
	l.Init = init
	return l
}

func (l *LoadOrInitFunc[T]) LoadOrInit(k any) (actual T, loaded bool) {
	l.l.Lock()
	defer l.l.Unlock()
	a, b := l.loadOrInitP(k)
	return *a, b
}

func (l *LoadOrInitFunc[T]) LoadOrInitPThen(k any) func(func(actual *T, loaded bool) (*T, bool)) (*T, bool) {
	return func(f func(actual *T, loaded bool) (*T, bool)) (*T, bool) {
		l.l.Lock()
		defer l.l.Unlock()
		return f(l.loadOrInitP(k))
	}
}

func (l *LoadOrInitFunc[T]) LoadOrInitP(k any) (actual *T, loaded bool) {
	l.l.Lock()
	defer l.l.Unlock()

	return l.loadOrInitP(k)
}

func (l *LoadOrInitFunc[T]) loadOrInitP(k any) (actual *T, loaded bool) {
	if l.cache == nil {
		l.cache = l.Init()
	}
	if actual, loaded := l.Map.LoadOrStore(k, l.cache); !loaded {
		l.cache = nil
		return actual.(*T), false
	} else {
		return actual.(*T), true
	}
}
