package ctx

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	ptr       = &struct{}{}
	ErrWaitTo = errors.New("ErrWaitTo")
)

type Ctx struct {
	Ctx context.Context
	i32 *atomic.Int32
}

/*
ctx,done := WithWait(ctx, time.Second)

	go func(){
			done1 := Wait(ctx)
			defer done1()
			do something..
	}()

done()// wait done1 or after one second
*/
func WithWait(sctx context.Context, to ...time.Duration) (dctx context.Context, done func() error) {
	if ctxp, ok := sctx.Value(ptr).(*Ctx); ok {
		ctxp.i32.Add(1)
	}

	ctx := &Ctx{i32: &atomic.Int32{}}
	ctx.Ctx = context.WithValue(sctx, ptr, ctx)

	dctx = ctx.Ctx
	done = func() error {
		<-ctx.Ctx.Done()
		if ctxp, ok := sctx.Value(ptr).(*Ctx); ok {
			defer ctxp.i32.Add(-1)
		}
		be := time.Now()
		for !ctx.i32.CompareAndSwap(0, -1) {
			if len(to) > 0 && time.Since(be) > to[0] {
				return ErrWaitTo
			}
			runtime.Gosched()
		}
		return nil
	}
	return
}

/*
	func(ctx context.Context){
			done := Wait(ctx)
			defer done()
			do something..
	}
*/
func Wait(ctx context.Context) (done func()) {
	if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
		ctxp.i32.Add(1)
	}
	<-ctx.Done()
	return func() {
		if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
			ctxp.i32.Add(-1)
		}
	}
}

/*
	func(ctx context.Context){
		ctx, done := WaitCtx(ctx)
		defer done()

		do something..
		select {
			case <-ctx:
			defualt:
		}
		do something..
	}
*/
func WaitCtx(ctx context.Context) (dctx context.Context, done func()) {
	if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
		ctxp.i32.Add(1)
	}
	return ctx, func() {
		if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
			ctxp.i32.Add(-1)
		}
	}
}

func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
