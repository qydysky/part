package ctx

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	ptr            = &struct{}{}
	ErrWaitTo      = errors.New("ErrWaitTo")
	ErrNothingWait = errors.New("ErrNothingWait")
)

type Ctx struct {
	Ctx context.Context
	w32 *atomic.Int32
	r32 *atomic.Int32
}

// planNum 可以为0 表示无法预知的调用次数，如果在mainDone调用前没有Wait、WithWait时，mainDone将返回ErrNothingWait
//
//		mainCtx, mainDone := WithWait(ctx, 0, time.Second)
//		defer mainDone()// wait done1 or after one second
//
//		go func(){
//			ctx1, done1 := WaitCtx(mainCtx)
//			defer done1()
//			do something..
//	 		<-ctx1.Done() // wait mainDone call
//		}()
func WithWait(sctx context.Context, planNum int32, to ...time.Duration) (dctx context.Context, done func() error) {
	if ctxp, ok := sctx.Value(ptr).(*Ctx); ok {
		ctxp.r32.Add(1)
		ctxp.w32.Add(-1)
	}

	ctx := &Ctx{w32: &atomic.Int32{}, r32: &atomic.Int32{}}
	ctx.Ctx = context.WithValue(sctx, ptr, ctx)
	ctx.w32.Add(planNum)

	var doneWait context.CancelFunc
	dctx, doneWait = context.WithCancel(ctx.Ctx)
	done = func() error {
		doneWait()
		if ctxp, ok := sctx.Value(ptr).(*Ctx); ok {
			defer func() {
				ctxp.r32.Add(-1)
			}()
		}
		if planNum == 0 && ctx.w32.Load() == 0 {
			return ErrNothingWait
		}
		be := time.Now()
		for ctx.w32.Load() > 0 {
			if len(to) > 0 && time.Since(be) > to[0] {
				return ErrWaitTo
			}
			runtime.Gosched()
		}
		for !ctx.r32.CompareAndSwap(0, -1) {
			if len(to) > 0 && time.Since(be) > to[0] {
				return ErrWaitTo
			}
			runtime.Gosched()
		}
		if len(to) > 0 && time.Since(be) > to[0] {
			return ErrWaitTo
		}
		return nil
	}
	return
}

//	go func(){
//		ctx1, done1 := WaitCtx(mainCtx)
//		defer done1()
//		do something..
//		<-ctx1.Done() // wait mainDone call
//	}()
func WaitCtx(ctx context.Context) (dctx context.Context, done func()) {
	if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
		ctxp.r32.Add(1)
		ctxp.w32.Add(-1)
	}
	return ctx, func() {
		if ctxp, ok := ctx.Value(ptr).(*Ctx); ok {
			ctxp.r32.Add(-1)
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

type Value[T any] struct {
	data T
}

func (t *Value[T]) get() T {
	return t.data
}

func (t *Value[T]) set(data T) {
	t.data = data
}

func (t *Value[T]) linkCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, t, t)
}

func putVal[T any](ctx context.Context, key *Value[T], v T) {
	if pt, ok := ctx.Value(key).(*Value[T]); ok {
		pt.set(v)
	} else {
		panic("")
	}
}
