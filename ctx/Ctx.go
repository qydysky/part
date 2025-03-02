package ctx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const sleepDru = 100

var (
	ptr            = &struct{}{}
	ErrWaitTo      = errors.New("ErrWaitTo")
	ErrNothingWait = errors.New("ErrNothingWait")
	ErrDoneCalled  = errors.New("ErrDoneCalled")
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

	var oncef atomic.Bool
	done = func() error {
		if !oncef.CompareAndSwap(false, true) {
			return ErrDoneCalled
		}

		doneWait()
		if ctxp, ok := sctx.Value(ptr).(*Ctx); ok {
			defer ctxp.r32.Add(-1)
		}
		if planNum == 0 && ctx.w32.Load() == 0 {
			return ErrNothingWait
		}
		be := time.Now()
		for ctx.w32.Load() > 0 {
			if len(to) > 0 && time.Since(be) > to[0] {
				return ErrWaitTo
			}
			time.Sleep(time.Millisecond * sleepDru)
			// runtime.Gosched()
		}
		for !ctx.r32.CompareAndSwap(0, -1) {
			if len(to) > 0 && time.Since(be) > to[0] {
				return ErrWaitTo
			}
			time.Sleep(time.Millisecond * sleepDru)
			// runtime.Gosched()
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
//
// or
// use as a normal context.WithCancel(ctx)
func WaitCtx(ctx context.Context) (dctx context.Context, done func()) {
	dctx1, done1 := context.WithCancel(ctx)
	if ctxp, ok := dctx1.Value(ptr).(*Ctx); ok {
		ctxp.r32.Add(1)
		ctxp.w32.Add(-1)
	}
	return dctx1, sync.OnceFunc(func() {
		done1()
		if ctxp, ok := dctx1.Value(ptr).(*Ctx); ok {
			ctxp.r32.Add(-1)
		}
	})
}

func Done(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

// errCtx := pctx.Value[error]{}
//
// cancelC = errCtx.LinkCtx(cancelC)
//
// pctx.PutVal(cancelC, &errCtx, fmt.Errorf("%vs未接收到有效数据", readTO))
//
// err := errCtx.Get()
type Value[T any] struct {
	data T
}

func (t *Value[T]) Get() T {
	return t.data
}

func (t *Value[T]) Set(data T) {
	t.data = data
}

func (t *Value[T]) LinkCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, t, t)
}

func PutVal[T any](ctx context.Context, key *Value[T], v T) {
	if pt, ok := ctx.Value(key).(*Value[T]); ok {
		pt.Set(v)
	}
}

var (
	selfCancel     = "selfCancel"
	ErrNotCarryYet = errors.New("ErrNotCarryYet")
)

func CarryCancel(ctx context.Context, cancelFunc context.CancelFunc) context.Context {
	return context.WithValue(ctx, &selfCancel, cancelFunc)
}

func CallCancel(ctx context.Context) error {
	if pt, ok := ctx.Value(&selfCancel).(context.CancelFunc); ok {
		pt()
	} else {
		return ErrNotCarryYet
	}
	return nil
}

func GenTOCtx(t time.Duration) context.Context {
	return CarryCancel(context.WithTimeout(context.Background(), t))
}

func GenDLCtx(t time.Time) context.Context {
	return CarryCancel(context.WithDeadline(context.Background(), t))
}
