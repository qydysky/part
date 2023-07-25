package part

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var (
	ErrWrongType = errors.New("ErrWrongType")
)

type CItem struct {
	Key  string
	Deal func(ctx context.Context, ptr any) error
}

func NewCI[T any](key string, f func(ctx context.Context, ptr *T) error) *CItem {
	return &CItem{
		Key: key,
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

type components struct {
	MatchF func(mkey, key string) bool

	m  []*CItem
	mm map[string]int
	sync.RWMutex
}

// strings.HasPrefix
//
// strings.HasSuffix
//
// DotMatch
func NewComp(matchf func(mkey, key string) bool) *components {
	return &components{
		MatchF: matchf,
		mm:     make(map[string]int),
	}
}

func (t *components) Put(item *CItem) error {
	t.Lock()
	defer t.Unlock()
	if _, ok := t.mm[item.Key]; ok {
		return errors.Join(ErrConflict, errors.New(item.Key))
	}
	t.m = append(t.m, item)
	sort.Slice(t.m, func(i, j int) bool {
		return t.m[i].Key < t.m[j].Key
	})
	for i := 0; i < len(t.m); i++ {
		t.mm[t.m[i].Key] = i
	}
	return nil
}

func (t *components) Del(key string) {
	t.Lock()
	for i := 0; i < len(t.m); i++ {
		if t.MatchF(t.m[i].Key, key) {
			delete(t.mm, t.m[i].Key)
			t.m = append(t.m[:i], t.m[i+1:]...)
			i -= 1
		}
	}
	for i := 0; i < len(t.m); i++ {
		t.mm[t.m[i].Key] = i
	}
	t.Unlock()
}

func (t *components) Run(key string, ctx context.Context, ptr any) error {
	t.RLock()
	defer t.RUnlock()

	var (
		i   = 0
		got = false
	)

	if mi, ok := t.mm[key]; ok {
		i = mi
	}
	for ; i < len(t.m) && t.MatchF(t.m[i].Key, key); i++ {
		got = true
		if e := t.m[i].Deal(ctx, ptr); e != nil {
			return errors.Join(ErrCItemErr, e)
		}
	}
	if !got {
		return errors.Join(ErrNotExist, errors.New(key))
	}

	return nil
}

func Put[T any](key string, f func(ctx context.Context, ptr *T) error) error {
	return Comp.Put(&CItem{
		Key: key,
		Deal: func(ctx context.Context, ptr any) error {
			if item, ok := ptr.(*T); ok {
				return f(ctx, item)
			}
			return errors.Join(ErrWrongType, errors.New(key))
		},
	})
}

func Run[T any](key string, ctx context.Context, ptr *T) error {
	return Comp.Run(key, ctx, ptr)
}

func Init(f func(mkey, key string) bool) {
	Comp = NewComp(f)
}

func DotMatch(mkey, key string) bool {
	return mkey == key || (len(mkey) > len(key) && mkey[0:len(key)] == key && mkey[len(key)] == '.')
}

var Comp *components = NewComp(DotMatch)
