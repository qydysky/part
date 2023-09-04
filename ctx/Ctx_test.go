package ctx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	ctx1, done := WithWaitTo(ctx, time.Second)
	go func() {
		done := ctx1.Wait()
		defer done()
	}()
	if done() != nil {
		t.Fatal()
	}
}

func TestMain2(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	ctx1, done := WithWaitTo(ctx, time.Second)
	go func() {
		done := ctx1.Wait()
		time.Sleep(time.Second * 2)
		defer done()
	}()
	time.Sleep(time.Second)
	if e := done(); !errors.Is(e, ErrWaitTo) {
		t.Fatal(e)
	}
}

func TestMain3(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	ctx1, done := WithWaitTo(ctx, time.Second)
	go func() {
		ctx2, done := WithWaitTo(ctx1, time.Second)
		go func() {
			done := ctx2.Wait()
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
	ctx1, done := WithWaitTo(ctx, time.Second)
	go func() {
		ctx2, done := WithWaitTo(ctx1, time.Second)
		go func() {
			done := ctx2.Wait()
			time.Sleep(time.Second * 2)
			defer done()
		}()
		if e := done(); !errors.Is(e, ErrWaitTo) {
			t.Fail()
		}
	}()
	time.Sleep(time.Second)
	if e := done(); !errors.Is(e, ErrWaitTo) {
		t.Fail()
	}
}
