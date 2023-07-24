package part

import (
	"context"
	"errors"
	"testing"
)

func Test1(t *testing.T) {
	Put(`1`, false, func(ctx context.Context, ptr *int) error {
		if *ptr > 1 {
			return nil
		}
		return errors.New("1")
	})

	if e := Put(`1`, false, func(ctx context.Context, ptr *int) error {
		if *ptr > 1 {
			return nil
		}
		return errors.New("1")
	}); e != ErrConflict {
		t.Fatal()
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

	if e := Comp.Run(`1.2.1`, context.Background(), &s); e != ErrNotExist {
		t.Fatal(e)
	}

	if e := Comp.Run(`1`, context.Background(), &s); e != nil {
		t.Fatal(e)
	}
}

func Benchmark2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Put[int](`1`, false, func(ctx context.Context, ptr *int) error {
			return nil
		})
	}
}

func Benchmark1(b *testing.B) {
	Put[int](`1`, false, func(ctx context.Context, ptr *int) error {
		return nil
	})
	Put[int](`1.2`, false, func(ctx context.Context, ptr *int) error {
		return nil
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if e := Run(`1.2`, context.Background(), &i); e != nil {
			b.Fatal()
		}
	}
}
