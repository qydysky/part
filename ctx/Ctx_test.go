package ctx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	ctx1, done := WithWait(context.Background(), 1, time.Second*2)
	t0 := time.Now()
	go func() {
		ctx2, done1 := WaitCtx(ctx1)
		defer done1()
		<-ctx2.Done()
		if time.Since(t0) < time.Millisecond*100 {
			t.Fail()
		}
		time.Sleep(time.Second)
	}()
	time.Sleep(time.Second)
	t1 := time.Now()
	if done() != nil {
		t.Fatal()
	}
	if time.Since(t1) < time.Second {
		t.Fail()
	}
}

func TestMain5(t *testing.T) {
	ctx1, done := WithWait(context.Background(), 1, time.Second*2)
	go func() {
		ctx2, done1 := WaitCtx(ctx1)
		go func() {
			time.Sleep(time.Millisecond * 500)
			done1()
		}()
		t1 := time.Now()
		<-ctx2.Done()
		if time.Since(t1)-time.Millisecond*500 > time.Millisecond*100 {
			t.Fatal()
		}
		time.Sleep(time.Second)
	}()
	time.Sleep(time.Second)
	t1 := time.Now()
	if done() != nil {
		t.Fatal()
	}
	if time.Since(t1) > time.Second {
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
	ctx = val.LinkCtx(ctx)
	PutVal(ctx, &val, errors.New("aaa"))
	if val.Get().Error() != "aaa" {
		t.Fatal()
	}
}

func TestMain4(t *testing.T) {
	ctx := CarryCancel(context.WithCancel(context.Background()))
	time.AfterFunc(time.Millisecond*500, func() {
		if CallCancel(ctx) != nil {
			t.Fail()
		}
	})
	n := time.Now()
	<-ctx.Done()
	if time.Since(n) < time.Millisecond*500 {
		t.Fail()
	}
}
