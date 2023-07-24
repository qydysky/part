package part

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrWrongType = errors.New("ErrWrongType")
)

type CItem struct {
	Key  string
	Lock bool
	Deal func(ctx context.Context, ptr any) error
	l    sync.Mutex
}

func (t *CItem) deal(ctx context.Context, ptr any) error {
	t.l.Lock()
	defer t.l.Unlock()
	return t.Deal(ctx, ptr)
}

func NewCI[T any](key string, lock bool, f func(ctx context.Context, ptr *T) error) *CItem {
	return &CItem{
		Key:  key,
		Lock: lock,
		Deal: func(ctx context.Context, ptr any) error {
			if item, ok := ptr.(*T); ok {
				return f(ctx, item)
			}
			return ErrWrongType
		},
	}
}

var (
	ErrNotExist = errors.New("ErrNotExist")
	ErrCItemErr = errors.New("ErrCItemErr")
	ErrConflict = errors.New("ErrConflict")
)

type Components struct {
	m  []*CItem
	mm map[string]int
	sync.RWMutex
}

func (t *Components) Put(item *CItem) error {
	t.Lock()
	defer t.Unlock()
	for i := 0; i < len(t.m); i++ {
		if t.m[i].Key == item.Key {
			return ErrConflict
		}
	}
	t.m = append(t.m, item)
	sort.Slice(t.m, func(i, j int) bool {
		return t.m[i].Key < t.m[j].Key
	})
	if t.mm == nil {
		t.mm = make(map[string]int)
	}
	for i := 0; i < len(t.m); i++ {
		if strings.HasPrefix(item.Key, t.m[i].Key) {
			t.mm[t.m[i].Key] = i
		}
	}
	return nil
}

func (t *Components) Del(key string) {
	t.Lock()
	for i := 0; i < len(t.m); i++ {
		if strings.HasPrefix(t.m[i].Key, key) {
			delete(t.mm, t.m[i].Key)
			t.m = append(t.m[:i], t.m[i+1:]...)
			i -= 1
		}
	}
	t.Unlock()
}

func (t *Components) Run(key string, ctx context.Context, ptr any) error {
	t.RLock()
	defer t.RUnlock()

	if i, ok := t.mm[key]; ok {
		for ; i < len(t.m) && strings.HasPrefix(t.m[i].Key, key); i++ {
			if e := t.m[i].deal(ctx, ptr); e != nil {
				return errors.Join(ErrCItemErr, e)
			}
		}
	} else {
		return ErrNotExist
	}

	return nil
}

func Put[T any](key string, lock bool, f func(ctx context.Context, ptr *T) error) error {
	return Comp.Put(&CItem{
		Key:  key,
		Lock: lock,
		Deal: func(ctx context.Context, ptr any) error {
			if item, ok := ptr.(*T); ok {
				return f(ctx, item)
			}
			return ErrWrongType
		},
	})
}

func Run[T any](key string, ctx context.Context, ptr *T) error {
	return Comp.Run(key, ctx, ptr)
}

var Comp Components
