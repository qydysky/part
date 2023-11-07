package ctx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	ctx1, done := WithWait(context.Background(), 1, time.Second)
	t0 := time.Now()
	go func() {
		ctx2, done1 := WaitCtx(ctx1)
		defer done1()
		<-ctx2.Done()
		if time.Since(t0) < time.Millisecond*100 {
			t.Fail()
		}
	}()
	time.Sleep(time.Second)
	if done() != nil {
		t.Fatal()
	}
}

func TestMain1(t *testing.T) {
	ctx1, done := WithWait(context.Background(), 1, time.Second)
	t0 := time.Now()
	go func() {
		ctx2, _ := WaitCtx(ctx1)
		<-ctx2.Done()
		if time.Since(t0) < time.Millisecond*100 {
			t.Fail()
		}
	}()
	time.Sleep(time.Second)
	if !errors.Is(done(), ErrWaitTo) {
		t.Fatal()
	}
}

func TestMain2(t *testing.T) {
	ctx1, done := WithWait(context.Background(), 0, time.Second)
	t0 := time.Now()
	go func() {
		time.Sleep(time.Second)
		ctx2, done := WaitCtx(ctx1)
		defer done()
		<-ctx2.Done()
		if time.Since(t0) < time.Millisecond*100 {
			t.Fail()
		}
	}()
	if !errors.Is(done(), ErrNothingWait) {
		t.Fatal()
	}
}

func TestMain3(t *testing.T) {
	ctx := context.Background()
	val := Value[error]{}
	ctx = val.linkCtx(ctx)
	putVal(ctx, &val, errors.New("aaa"))
	if val.get().Error() != "aaa" {
		t.Fatal()
	}
}
