// Deprecaated: use component2
package part

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	ErrStopRun = errors.New("ErrStopRun")
	ErrSelfDel = errors.New("ErrSelfDel")
)

type Component[T, E any] struct {
	del  atomic.Bool
	deal func(ctx context.Context, ptr T) (E, error)
}

func NewComp[T, E any](deal func(ctx context.Context, ptr T) (ret E, err error)) *Component[T, E] {
	return &Component[T, E]{atomic.Bool{}, deal}
}

func (t *Component[T, E]) Run(ctx context.Context, ptr T) (ret E, err error) {
	if t.del.Load() || t.deal == nil {
		return
	}
	return t.deal(ctx, ptr)
}

func (t *Component[T, E]) Del() {
	t.del.Store(true)
}

// type Components[T any] struct {
// 	lock  sync.RWMutex
// 	comps []*Component[T]
// }

// func NewComps[T any](c ...*Component[T]) *Components[T] {
// 	return &Components[T]{comps: c}
// }

// func (t *Components[T]) Put(c ...*Component[T]) {
// 	t.lock.Lock()
// 	t.comps = append(t.comps, c...)
// 	t.lock.Unlock()
// }

// func (t *Components[T]) Del(c ...*Component[T]) {
// 	t.lock.Lock()
// 	for i := 0; i < len(t.comps); i++ {
// 		for j := 0; j < len(c); j++ {
// 			if t.comps[i] == c[j] {
// 				copy(t.comps[i:], t.comps[i+1:])
// 				t.comps = t.comps[:len(t.comps)-1]
// 				copy(c[i:], c[i+1:])
// 				c = c[:len(c)-1]
// 				break
// 			}
// 		}
// 	}
// 	t.lock.Unlock()
// }

// func (t *Components[T]) DelAll() {
// 	t.lock.Lock()
// 	clear(t.comps)
// 	t.lock.Unlock()
// }

// func (t *Components[T]) Run(ctx context.Context, ptr T) error {
// 	var needDel bool

// 	t.lock.RLock()
// 	defer func() {
// 		t.lock.RUnlock()
// 		if needDel {
// 			t.lock.Lock()
// 			for i := 0; i < len(t.comps); i++ {
// 				if t.comps[i].del.Load() {
// 					copy(t.comps[i:], t.comps[i+1:])
// 					t.comps = t.comps[:len(t.comps)-1]
// 				}
// 			}
// 			t.lock.Unlock()
// 		}
// 	}()

// 	for i := 0; i < len(t.comps); i++ {
// 		if t.comps[i].del.Load() || t.comps[i].deal == nil {
// 			continue
// 		}
// 		e := t.comps[i].deal(ctx, ptr)
// 		if errors.Is(e, ErrSelfDel) {
// 			t.comps[i].del.Store(true)
// 			needDel = true
// 		}
// 		if errors.Is(e, ErrStopRun) {
// 			return e
// 		}
// 	}

// 	return nil
// }

// func (t *Components[T]) Start(ctx context.Context, ptr T, concurrency ...int) error {
// 	var needDel bool

// 	t.lock.RLock()
// 	defer func() {
// 		t.lock.RUnlock()
// 		if needDel {
// 			t.lock.Lock()
// 			for i := 0; i < len(t.comps); i++ {
// 				if t.comps[i].del.Load() {
// 					copy(t.comps[i:], t.comps[i+1:])
// 					t.comps = t.comps[:len(t.comps)-1]
// 				}
// 			}
// 			t.lock.Unlock()
// 		}
// 	}()

// 	var (
// 		wg  sync.WaitGroup
// 		con chan struct{}
// 		err = make(chan error, len(t.comps))
// 	)
// 	if len(concurrency) > 0 {
// 		con = make(chan struct{}, concurrency[0])
// 	}
// 	wg.Add(len(t.comps))

// 	for i := 0; i < len(t.comps); i++ {
// 		if t.comps[i].del.Load() || t.comps[i].deal == nil {
// 			wg.Done()
// 			err <- nil
// 			continue
// 		}
// 		if con != nil {
// 			con <- struct{}{}
// 		}
// 		go func(i int) {
// 			e := t.comps[i].deal(ctx, ptr)
// 			if errors.Is(e, ErrSelfDel) {
// 				t.comps[i].del.Store(true)
// 			}
// 			err <- e
// 			wg.Done()
// 			if con != nil {
// 				<-con
// 			}
// 		}(i)
// 	}

// 	wg.Wait()

// 	for {
// 		select {
// 		case e := <-err:
// 			if errors.Is(e, ErrSelfDel) {
// 				needDel = true
// 			} else if e != nil {
// 				return e
// 			}
// 		default:
// 			return nil
// 		}
// 	}
// }
