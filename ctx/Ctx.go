package ctx

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	ErrWaitTo = errors.New("ErrWaitTo")
	ErrToLess = errors.New("ErrToLess")
)

type Ctx struct {
	Ctx context.Context
	i32 *atomic.Int32
	to  time.Duration
}

// ctx,done := WithWaitTo(..)
//
//	go func(){
//			done1 := ctx.Wait()
//			defer done1()
//	}()
//
// done()// wait done1
func WithWaitTo(sctx context.Context, to time.Duration) (ctx *Ctx, done func() error) {
	if ctx, ok := sctx.(*Ctx); ok {
		if ctx.to < to {
			panic(ErrToLess)
		}
		ctx.i32.Add(1)
	}
	ctx = &Ctx{Ctx: sctx, i32: &atomic.Int32{}, to: to}
	done = func() error {
		<-ctx.Ctx.Done()
		if ctx, ok := sctx.(*Ctx); ok {
			defer ctx.i32.Add(-1)
		}
		be := time.Now()
		for !ctx.i32.CompareAndSwap(0, -1) {
			if time.Since(be) > to {
				return ErrWaitTo
			}
			runtime.Gosched()
		}
		return nil
	}
	return
}

func (t Ctx) Deadline() (deadline time.Time, ok bool) {
	return t.Ctx.Deadline()
}

func (t Ctx) Done() <-chan struct{} {
	return t.Ctx.Done()
}

func (t Ctx) Err() error {
	return t.Ctx.Err()
}

func (t Ctx) Value(key any) any {
	return t.Ctx.Value(key)
}

func (t Ctx) Wait() (done func()) {
	t.i32.Add(1)
	<-t.Ctx.Done()
	return func() {
		t.i32.Add(-1)
	}
}
