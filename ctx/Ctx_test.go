package ctx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	ctx1, done := WithWait(ctx, 1, time.Second)
	go func() {
		done := Wait(ctx1)
		defer done()
	}()
	if done() != nil {
		t.Fatal()
	}
}

func TestMain1(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	ctx1, done := WithWait(ctx, 0, time.Second)
	go func() {
		done := Wait(ctx1)
		defer done()
		time.Sleep(100 * time.Millisecond)
	}()
	if e := done(); !errors.Is(e, ErrNothingWait) {
		t.Fatal(e)
	}
}

func TestMain2(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	ctx1, done := WithWait(ctx, 1, time.Second)
	go func() {
		done := Wait(ctx1)
		time.Sleep(time.Second * 2)
		defer done()
	}()
	if e := done(); !errors.Is(e, ErrWaitTo) {
		t.Fatal(e)
	}
}

func TestMain3(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	ctx1, done := WithWait(ctx, 1, time.Second)
	go func() {
		ctx2, done := WithWait(ctx1, 1, time.Second)
		go func() {
			done := Wait(ctx2)
			defer done()
		}()
		if done() != nil {
			t.Fail()
		}
	}()
	time.Sleep(time.Second)
	if done() != nil {
		t.Fatal()
	}
}

func TestMain4(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	ctx1, done := WithWait(ctx, 1, time.Second)
	go func() {
		ctx2, done := WithWait(ctx1, 1, time.Second)
		go func() {
			done := Wait(ctx2)
			time.Sleep(time.Second * 2)
			defer done()
		}()
		if e := done(); !errors.Is(e, ErrWaitTo) {
			t.Fail()
		}
	}()
	if e := done(); !errors.Is(e, ErrWaitTo) {
		t.Fatal(e)
	}
}

func TestMain5(t *testing.T) {
	ctx, can := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer can()

	ctx, done := WithWait(ctx, 1)

	var gr = func(ctx context.Context, to time.Duration) {
		done := Wait(ctx)
		defer done()
		time.Sleep(to)
	}

	bg := time.Now()

	go gr(ctx, 0)
	// go gr(ctx, time.Second)
	// go gr(ctx, time.Second*5)

	t.Log(done(), time.Since(bg))
}
