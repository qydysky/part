package part

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func Test1(t *testing.T) {
	Init(DotMatch)

	Put(`1`, func(ctx context.Context, ptr *int) error {
		if *ptr > 1 {
			return nil
		}
		return errors.New("1")
	})

	if e := Put(`1`, func(ctx context.Context, ptr *int) error {
		if *ptr > 1 {
			return nil
		}
		return errors.New("1")
	}); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	Comp.Put(&CItem{
		Key: `1.2`,
		Deal: func(_ context.Context, ptr any) error {
			if sp, ok := ptr.(*int); ok && *sp >= 2 {
				return nil
			}
			return errors.New("1.2")
		},
	})

	Comp.Put(&CItem{
		Key: `1.2.1`,
		Deal: func(_ context.Context, ptr any) error {
			if sp, ok := ptr.(*int); ok && *sp >= 3 {
				return nil
			}
			return errors.New("1.2.1")
		},
	})

	var s = 3
	if e := Comp.Run(`1`, context.Background(), &s); e != nil {
		t.Fatal(e)
	}

	Comp.Del(`1.2`)

	for i := 0; i < len(Comp.m); i++ {
		t.Log(Comp.m[i])
	}
	if e := Comp.Run(`1.2.1`, context.Background(), &s); e == nil {
		t.Fatal()
	}

	if e := Comp.Run(`1`, context.Background(), &s); e != nil {
		t.Fatal(e)
	}
}

func TestDot(t *testing.T) {
	Init(DotMatch)
	Put[int](`1`, func(ctx context.Context, ptr *int) error {
		if *ptr == 1 {
			return nil
		} else {
			return errors.New("1")
		}
	})
	Put[int](`12`, func(ctx context.Context, ptr *int) error {
		return errors.New("12")
	})
	Put[int](`1.2`, func(ctx context.Context, ptr *int) error {
		return errors.New("1.2")
	})
	i := 1
	if e := Run(`1`, context.Background(), &i); !strings.Contains(e.Error(), "1.2") {
		t.Fatal(e)
	}
}

func Test3(t *testing.T) {
	Init(DotMatch)
	sumup := func(ctx context.Context, ptr *int) error {
		return nil
	}
	if e := Put[int](`bili_danmu.Reply.wsmsg.preparing.sumup`, sumup); e != nil {
		panic(e)
	} else {
		println("bili_danmu.Reply.wsmsg.preparing.sumup")
	}
	i := 1
	if e := Run(`bili_danmu.Reply.wsmsg.preparing`, context.Background(), &i); e != nil {
		t.Fatal(e)
	}
}

func Benchmark2(b *testing.B) {
	Init(DotMatch)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Put[int](strconv.Itoa(i), func(ctx context.Context, ptr *int) error {
			return nil
		})
	}
}

func Benchmark1(b *testing.B) {
	Init(DotMatch)
	for i := 0; i < 1000; i++ {
		Put[int](strconv.Itoa(i), func(ctx context.Context, ptr *int) error {
			return nil
		})
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if e := Run(`1`, ctx, &i); e != nil {
			b.Fatal(e)
		}
	}
}
