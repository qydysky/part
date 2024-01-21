package part

import (
	"context"
	"fmt"
	"testing"
)

// func Test1(t *testing.T) {
// 	Comp = NewComp()
// 	Put(`1`, func(ctx context.Context, ptr *int) error {
// 		if *ptr > 1 {
// 			return nil
// 		}
// 		return errors.New("1")
// 	})

// 	if e := Put(`1`, func(ctx context.Context, ptr *int) error {
// 		if *ptr > 1 {
// 			return nil
// 		}
// 		return errors.New("1")
// 	}); !errors.Is(e, ErrConflict) {
// 		t.Fatal(e)
// 	}

// 	Comp.Put(`1.2`, func(_ context.Context, ptr any) error {
// 		if sp, ok := ptr.(*int); ok && *sp >= 2 {
// 			return nil
// 		}
// 		return errors.New("1.2")
// 	})

// 	Comp.Put(`1.2.1`, func(_ context.Context, ptr any) error {
// 		if sp, ok := ptr.(*int); ok && *sp >= 3 {
// 			return nil
// 		}
// 		return errors.New("1.2.1")
// 	})

// 	if e := Comp.Link(map[string][]string{
// 		`1`: {`1`},
// 	}); e != nil {
// 		t.Fatal(e)
// 	}

// 	var s = 3
// 	if e := Comp.Run(`1`, context.Background(), &s); e != nil {
// 		t.Fatal(e)
// 	}

// 	Comp.Del(`1.2`)

// 	if e := Comp.Run(`1.2.1`, context.Background(), &s); e == nil {
// 		t.Fatal()
// 	}

// 	if e := Comp.Run(`1`, context.Background(), &s); e != nil {
// 		t.Fatal(e)
// 	}
// }

// func TestDot(t *testing.T) {
// 	Comp = NewComp()
// 	Put[int](`1`, func(ctx context.Context, ptr *int) error {
// 		if *ptr == 1 {
// 			return nil
// 		} else {
// 			return errors.New("1")
// 		}
// 	})
// 	Put[int](`12`, func(ctx context.Context, ptr *int) error {
// 		return errors.New("12")
// 	})
// 	Put[int](`1.2`, func(ctx context.Context, ptr *int) error {
// 		return errors.New("1.2")
// 	})
// 	Link(map[string][]string{
// 		`1`: {`1.2`},
// 	})
// 	i := 1
// 	if e := Run(`1`, context.Background(), &i); !strings.Contains(e.Error(), "1.2") {
// 		t.Fatal(e)
// 	}
// }

// func Test3(t *testing.T) {
// 	Comp = NewComp()
// 	sumup := func(ctx context.Context, ptr *int) error {
// 		return nil
// 	}
// 	if e := Put[int](`bili_danmu.Reply.wsmsg.preparing.sumup`, sumup); e != nil {
// 		panic(e)
// 	} else {
// 		println("bili_danmu.Reply.wsmsg.preparing.sumup")
// 	}
// 	Link(map[string][]string{
// 		`bili_danmu.Reply.wsmsg.preparing`: {`bili_danmu.Reply.wsmsg.preparing.sumup`},
// 	})
// 	i := 1
// 	if e := Run(`bili_danmu.Reply.wsmsg.preparing`, context.Background(), &i); e != nil {
// 		t.Fatal(e)
// 	}
// }

// func Test4(t *testing.T) {
// 	type empty struct{}
// 	if pkg := Sign[empty](`1`, `2`); pkg != `github.com/qydysky/part/component.1.2` {
// 		t.Fatal(pkg)
// 	}
// }

// func Test5(t *testing.T) {
// 	t.Log(GetPkgSign())
// }

// func Benchmark2(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		Put[int](strconv.Itoa(i), func(ctx context.Context, ptr *int) error {
// 			return nil
// 		})
// 	}
// }

// func Benchmark1(b *testing.B) {
// 	for i := 0; i < 1000; i++ {
// 		Put[int](`1`, func(ctx context.Context, ptr *int) error {
// 			return nil
// 		})
// 	}
// 	Link(map[string][]string{
// 		`1`: {`1`},
// 	})
// 	ctx := context.Background()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		if e := Run(`1`, ctx, &i); e != nil {
// 			b.Fatal(e)
// 		}
// 	}
// }

func TestMain(t *testing.T) {
	c1 := NewComp(func(ctx context.Context, ptr string) (any, error) {
		fmt.Println(ptr)
		return nil, nil
	})
	c1.Run(context.Background(), "1")
}

// func TestMain2(t *testing.T) {
// 	c1 := NewComp(func(ctx context.Context, ptr string) (any,error) {
// 		fmt.Println(ptr)
// 		return ErrSelfDel
// 	})
// 	c2 := NewComp(func(ctx context.Context, ptr string) (any,error) {
// 		fmt.Println(ptr + "s")
// 		return nil
// 	})
// 	cs1 := NewComps[string]()
// 	cs1.Put(c1, c2)
// 	cs1.Run(context.Background(), "1")
// 	cs1.Start(context.Background(), "1")
// }
