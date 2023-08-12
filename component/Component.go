package part

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"sync/atomic"

	psync "github.com/qydysky/part/sync"
)

var (
	ErrNoLink    = errors.New("ErrNoLink")
	ErrLinked    = errors.New("ErrLinked")
	ErrNoExist   = errors.New("ErrNoExist")
	ErrConflict  = errors.New("ErrConflict")
	ErrWrongType = errors.New("ErrWrongType")
)

type components struct {
	m        psync.Map
	link     map[string][]string
	loadLink atomic.Bool
}

func NewComp() *components {
	return &components{link: make(map[string][]string)}
}

func (t *components) Put(Key string, Deal func(ctx context.Context, ptr any) error) error {
	_, loaded := t.m.LoadOrStore(Key, Deal)
	if loaded {
		return errors.Join(ErrConflict, errors.New(Key))
	}
	return nil
}

func (t *components) Del(Key string) {
	t.m.Delete(Key)
}

func (t *components) Run(key string, ctx context.Context, ptr any) error {
	if !t.loadLink.Load() {
		return ErrNoLink
	}
	links := t.link[key]
	if len(links) == 0 {
		return ErrNoExist
	}
	for i := 0; i < len(links); i++ {
		if deal, ok := t.m.LoadV(links[i]).(func(ctx context.Context, ptr any) error); ok {
			if e := deal(ctx, ptr); e != nil {
				return e
			}
		}
	}
	return nil
}

func (t *components) Link(link map[string][]string) error {
	if t.loadLink.CompareAndSwap(false, true) {
		t.link = maps.Clone(link)
	}
	return ErrLinked
}

func Put[T any](key string, deal func(ctx context.Context, ptr *T) error) error {
	return Comp.Put(key, func(ctx context.Context, ptr any) error {
		if item, ok := ptr.(*T); ok {
			return deal(ctx, item)
		}
		return errors.Join(ErrWrongType, errors.New(key))
	})
}

func Run[T any](key string, ctx context.Context, ptr *T) error {
	return Comp.Run(key, ctx, ptr)
}

func Link(link map[string][]string) error {
	return Comp.Link(link)
}

func PKG[T any](sign ...string) (pkg string) {
	pkg = reflect.TypeOf(*new(T)).PkgPath()
	for i := 0; i < len(sign); i++ {
		pkg += "." + sign[i]
	}
	return
}

var Comp *components = NewComp()
