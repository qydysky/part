package ctx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMain13(t *testing.T) {
	t.Parallel()
	ctx1, done1 := WithWait(context.Background(), 0, time.Second*2)
	go func() {
		ctx2, done2 := WaitCtx(ctx1)
		defer done2()
		time.AfterFunc(time.Millisecond*100, func() { done1() })
		<-ctx2.Done()
		time.Sleep(time.Millisecond * 200)
	}()
	time.Sleep(time.Millisecond * 200)
	if e := done1(true); e != nil {
		t.Fatal(e)
	}
}

func TestMain12(t *testing.T) {
	t.Parallel()
	ctx1, done1 := WithWait(context.Background(), 0, time.Second*2)
	ctx2, done2 := WithWait(ctx1, 0, time.Second*2)
	go func() {
		ctx3, done3 := WaitCtx(ctx2)
		defer done3()
		<-ctx3.Done()
	}()
	time.Sleep(time.Second)
	if e := done2(); e != nil {
		t.Fatal(e)
	}
	if e := done1(); e != nil {
		t.Fatal(e)
	}
}

func TestMain11(t *testing.T) {
	t.Parallel()
	ctx1, done1 := WithWait(context.Background(), 0, time.Second*2)
	ctx2, _ := WithWait(ctx1, 0, time.Second*2)
	go func() {
		ctx3, done3 := WaitCtx(ctx2)
		defer done3()
		<-ctx3.Done()
	}()
	time.Sleep(time.Second)
	if e := done1(); e == nil {
		t.Fatal(e)
	}
}

func TestMain10(t *testing.T) {
	t.Parallel()
	ctx1, done := WithWait(context.Background(), 0, time.Second*2)
	go func() {
		ctx2, done2 := WaitCtx(ctx1)
		_, done3 := WaitCtx(ctx2)
		defer done2()
		defer done3()
		<-ctx2.Done()
	}()
	time.Sleep(time.Second)
	if e := done(); e != nil {
		t.Fatal(e)
	}
}

func TestMain9(t *testing.T) {
	t.Parallel()
	ctx1, done := WithWait(context.Background(), 0, time.Second*2)
	go func() {
		ctx2, done2 := WaitCtx(ctx1)
		_, _ = WaitCtx(ctx2)
		defer done2()
		<-ctx2.Done()
	}()
	time.Sleep(time.Second)
	if e := done(); e == nil {
		t.Fatal(e)
	}
}

func TestMain(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	ctx1, done := WithWait(context.Background(), 1)
	t0 := time.Now()
	go func() {
		ctx2, done1 := WaitCtx(ctx1)
		defer done1()
		<-ctx2.Done()
		if time.Since(t0) < time.Millisecond*100 {
			t.Fail()
		}
		time.Sleep(time.Millisecond * 200)
	}()
	time.Sleep(time.Millisecond * 200)
	t1 := time.Now()
	if done() != nil {
		t.Fatal()
	}
	if time.Since(t1) < time.Millisecond*100 {
		t.Fail()
	}
	if !errors.Is(done(), ErrDoneCalled) {
		t.Fatal()
	}
}

func TestMain7(t *testing.T) {
	t.Parallel()
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

func TestMain6(t *testing.T) {
	t.Parallel()
	ctx1, done := WithWait(context.Background(), 1, time.Second*2)
	go func() {
		ctx2, done1 := WaitCtx(ctx1)
		go func() {
			time.Sleep(time.Millisecond * 500)
			done1()
		}()
		t1 := time.Now()
		<-ctx2.Done()
		done1()
		if time.Since(t1)-time.Millisecond*500 > time.Millisecond*100 {
			t.Fatal()
		}

		ctx3, done2 := WaitCtx(ctx1)
		defer done2()
		<-ctx3.Done()
		time.Sleep(time.Second)
	}()
	time.Sleep(time.Second)
	t1 := time.Now()
	if done() != nil {
		t.Fatal()
	}
	if time.Since(t1) < time.Second {
		t.Fatal()
	}
}

func TestMain1(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	ctx := context.Background()
	val := Value[error]{}
	ctx = val.LinkCtx(ctx)
	PutVal(ctx, &val, errors.New("aaa"))
	if val.Get().Error() != "aaa" {
		t.Fatal()
	}
}

func TestMain4(t *testing.T) {
	t.Parallel()
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
