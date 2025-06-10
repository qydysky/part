package part

import (
	"context"
	"errors"
	"fmt"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"

	pctx "github.com/qydysky/part/ctx"
	funcCtrl "github.com/qydysky/part/funcCtrl"
	psync "github.com/qydysky/part/sync"
)

// type test_item struct {
// 	data string
// }

// func Test_msgq(t *testing.T) {

// 	mq := New(5)
// 	mun := 100000
// 	mun_c := make(chan bool, mun)
// 	mun_s := make(chan bool, mun)

// 	var e int

// 	sig := mq.Sig()
// 	for i := 0; i < mun; i++ {
// 		go func() {
// 			mun_c <- true
// 			data, t0 := mq.Pull(sig)
// 			if o, ok := data.(string); o != `mmm` || !ok {
// 				e = 1
// 			}
// 			data1, _ := mq.Pull(t0)
// 			if o, ok := data1.(string); o != `mm1` || !ok {
// 				e = 2
// 			}
// 			mun_s <- true
// 		}()
// 	}

// 	for len(mun_c) != mun {
// 		t.Log(`>`, len(mun_c))
// 		sys.Sys().Timeoutf(1)
// 	}
// 	t.Log(`>`, len(mun_c))

// 	t.Log(`push mmm`)
// 	mq.Push(`mmm`)
// 	t.Log(`push mm1`)
// 	mq.Push(`mm1`)

// 	for len(mun_s) != mun {
// 		t.Log(`<`, len(mun_s))
// 		sys.Sys().Timeoutf(1)
// 	}
// 	t.Log(`<`, len(mun_s))

// 	if e != 0 {
// 		t.Error(e)
// 	}
// }

// func Test_msgq2(t *testing.T) {
// 	mq := New(5)

// 	mun_c := make(chan bool, 100)
// 	go func() {
// 		var (
// 			sig  = mq.Sig()
// 			data any
// 		)
// 		for {
// 			data, sig = mq.Pull(sig)
// 			if data.(test_item).data != `aa1` {
// 				t.Error(`1`)
// 			}
// 			mun_c <- true
// 		}
// 	}()
// 	go func() {
// 		var (
// 			sig  = mq.Sig()
// 			data any
// 		)
// 		for {
// 			data, sig = mq.Pull(sig)
// 			if data.(test_item).data != `aa1` {
// 				t.Error(`2`)
// 			}
// 			mun_c <- true
// 		}
// 	}()
// 	go func() {
// 		var (
// 			sig  = mq.Sig()
// 			data any
// 		)
// 		for {
// 			data, sig = mq.Pull(sig)
// 			if data.(test_item).data != `aa1` {
// 				t.Error(`3`)
// 			}
// 			mun_c <- true
// 		}
// 	}()
// 	var fin_turn = 0
// 	t.Log(`start`)
// 	time.Sleep(time.Second)
// 	for fin_turn < 1000000 {
// 		mq.Push(test_item{
// 			data: `aa1`,
// 		})
// 		<-mun_c
// 		<-mun_c
// 		<-mun_c
// 		fin_turn += 1
// 		fmt.Print("\r", fin_turn)
// 	}
// 	t.Log(`fin`)
// }

func BenchmarkXxx(b *testing.B) {
	mq := New()
	mq.Pull_tag(map[string]func(any) bool{
		`1`: func(_ any) bool {
			return false
		},
	})
	mq.Pull_tag(map[string]func(any) bool{
		`2`: func(_ any) bool {
			return false
		},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mq.Push_tag(`1`, nil)
		if i == b.N/2 {
			mq.Push_tag(`2`, nil)
		}
	}
}

func TestTORun(t *testing.T) {
	t.Parallel()
	panicC := make(chan any, 10)
	mq := New(time.Second, time.Second)
	mq.TOPanicFunc(func(a any) {
		panicC <- a
	})
	mq.Pull_tag_only(`test`, func(a any) (disable bool) {
		time.Sleep(time.Second * 10)
		return false
	})
	go mq.Push_tag(`test`, nil)
	e := <-panicC
	if !errors.Is(e.(error), ErrRunTO) {
		t.Fatal(e)
	}
}

func TestPushLock(t *testing.T) {
	t.Parallel()
	panicC := make(chan any, 10)
	mq := New(time.Second, time.Second*2)
	mq.TOPanicFunc(func(a any) {
		panicC <- a
	})
	mq.Pull_tag_only(`test`, func(a any) (disable bool) {
		mq.PushLock_tag(`lock`, nil)
		return false
	})
	go mq.Push_tag(`test`, nil)
	e := <-panicC
	if !errors.Is(e.(error), psync.ErrTimeoutToLock) {
		t.Fatal(e)
	}
}

func Benchmark_1(b *testing.B) {
	mq := New()
	mq.Pull_tag_only(`test`, func(a any) (disable bool) {
		return false
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mq.Push_tag(`test`, i)
	}
}

func Test_4(t *testing.T) {
	t.Parallel()
	mq := New()
	cancel := mq.Pull_tag(FuncMap{
		`del`: func(a any) (disable bool) {
			return true
		},
	})
	time.Sleep(time.Millisecond * 500)
	mq.PushLock_tag(`del`, nil)
	cancel()
}

func Test_2(t *testing.T) {
	t.Parallel()
	mq := New()
	cancel := mq.Pull_tag(FuncMap{
		`del`: func(a any) (disable bool) {
			t.Fatal()
			return false
		},
	})
	time.Sleep(time.Millisecond * 500)
	cancel()
	mq.PushLock_tag(`del`, nil)
}

func Test_5(t *testing.T) {
	t.Parallel()
	mq := New(time.Second, time.Second*2)
	mq.Pull_tag(FuncMap{
		`del`: func(a any) (disable bool) {
			t.Log(1)
			mq.Push_tag(`del1`, nil)
			t.Log(3)
			return false
		},
		`del1`: func(a any) (disable bool) {
			t.Log(2)
			return true
		},
	})
	mq.Push_tag(`del`, 1)
	mq.Push_tag(`del1`, 1)
}

func Test_1(t *testing.T) {
	t.Parallel()
	mq := New(time.Millisecond*5, time.Millisecond*10)
	go mq.Push_tag(`del`, nil)
	mq.Pull_tag(FuncMap{
		`del`: func(a any) (disable bool) {
			mq.Push_tag(`del1`, nil)
			return false
		},
		`del1`: func(a any) (disable bool) {
			return true
		},
	})
	time.Sleep(time.Millisecond * 500)
}

func Test_RemoveInPush(t *testing.T) {
	t.Parallel()
	mq := New(time.Second, time.Second*3)
	mq.Pull_tag(FuncMap{
		`r1`: func(a any) (disable bool) {
			mq.ClearAll()
			return true
		},
		`r2`: func(a any) (disable bool) {
			return true
		},
	})
	mq.PushLock_tag(`r1`, nil)
	if mq.funcs.Len() != 0 {
		t.Fatal()
	}
}

func Test_3(t *testing.T) {
	t.Parallel()
	mq := New(time.Millisecond*5, time.Millisecond*10)
	go mq.Push_tag(`sss`, nil)
	mq.Pull_tag(FuncMap{
		`test`: func(a any) (disable bool) {
			return false
		},
	})
	time.Sleep(time.Millisecond * 500)
}

func Test_Pull_tag_chan(t *testing.T) {
	t.Parallel()
	mq := New()
	ctx, cf := context.WithCancel(context.Background())
	_, ch := mq.Pull_tag_chan(`a`, 2, ctx)
	for i := 0; i < 5; i++ {
		mq.Push_tag(`a`, i)
	}
	if len(ch) != 1 {
		t.Fatal()
	}
	var o = 0
	for s := true; s; {
		select {
		case i := <-ch:
			o += i.(int)
		default:
			s = false
		}
	}
	if o != 4 {
		t.Fatal()
	}
	select {
	case <-ch:
		t.Fatal()
	default:
	}
	cf()
	mq.Push_tag(`a`, 1)
	select {
	case i := <-ch:
		if i != nil {
			t.Fatal()
		}
	default:
		t.Fatal()
	}
}

func Test_Pull_tag_chan2(t *testing.T) {
	t.Parallel()
	mq := New()

	mq.Pull_tag_chan(`a`, 1, context.Background())
	go func() {
		mq.PushLock_tag(`a`, nil)
		mq.PushLock_tag(`a`, nil)
	}()
}

func Test_msgq1(t *testing.T) {
	t.Parallel()
	mq := New()
	c := make(chan time.Time, 10)
	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(time.Time); ok {
				c <- v
				time.Sleep(time.Second)
			}
			return false
		},
	})

	{
		var w sync.WaitGroup
		w.Add(2)
		go func() {
			mq.Push_tag(`A1`, time.Now())
			w.Done()
		}()
		go func() {
			time.Sleep(time.Millisecond * 100)
			mq.Push_tag(`A1`, time.Now())
			w.Done()
		}()
		w.Wait()
		if t1 := time.Now().Add(-time.Second).Add(-time.Millisecond * 100).Sub(<-c).Milliseconds(); t1 > 50 {
			t.Fatal(t1)
		}
		if t1 := time.Now().Add(-time.Second).Sub(<-c).Milliseconds(); t1 > 50 {
			t.Fatal(t1)
		}
	}

	{
		var w sync.WaitGroup
		w.Add(2)
		go func() {
			mq.PushLock_tag(`A1`, time.Now())
			w.Done()
		}()
		go func() {
			time.Sleep(time.Millisecond * 100)
			mq.PushLock_tag(`A1`, time.Now())
			w.Done()
		}()
		w.Wait()
		if t1 := time.Now().Add(-2 * time.Second).Sub(<-c).Milliseconds(); t1 > 50 {
			t.Fatal(t1)
		}
		if t1 := time.Now().Add(-2 * time.Second).Add(-time.Millisecond * 100).Sub(<-c).Milliseconds(); t1 > 50 {
			t.Fatal(t1)
		}
	}
}

func Test_msgq9(t *testing.T) {
	t.Parallel()
	var mq = NewType[int]()
	var c = make(chan int, 10)
	mq.Pull_tag_only(`c`, func(i int) (disable bool) {
		c <- i
		return false
	})
	mq.PushLock_tag(`c`, 1)
	mq.ClearAll()
	mq.PushLock_tag(`c`, 2)
	t.Log(mq.m.funcs.Len())
	if l, i := len(c), <-c; l != 1 || i != 1 {
		t.Fatal(l, i)
	}

	fmt.Println(mq.m.allNeedRemove.Load())

	mq.Pull_tag(map[string]func(int) (disable bool){
		`c`: func(i int) (disable bool) {
			fmt.Println(1)
			c <- i
			return false
		},
		`s`: func(_ int) (disable bool) {
			mq.ClearAll()
			return false
		},
	})

	fmt.Println(mq.m.funcs.Len())

	mq.PushLock_tag(`c`, 1)
	mq.ClearAll()
	mq.PushLock_tag(`c`, 2)
	if l, i := len(c), <-c; l != 1 || i != 1 {
		t.Fatal(l, i)
	}
}

func Test_msgq2(t *testing.T) {
	t.Parallel()
	mq := New()

	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(bool); ok {
				return v
			}
			return false
		},
		`A3`: func(data any) bool {
			if v, ok := data.(int); ok && v > 50 {
				t.Fatal()
			}
			return false
		},
	})

	mq.Pull_tag(map[string]func(any) bool{
		`A2`: func(data any) bool {
			if v, ok := data.(bool); ok {
				return v
			}
			return false
		},
		`A3`: func(data any) bool {
			// if v, ok := data.(int); ok {
			// 	fmt.Println(`A2A3`, v)
			// }
			return false
		},
	})

	for i := 0; i < 1000; i++ {
		if i == 50 {
			mq.Push_tag(`A1`, true)
		}
		mq.Push_tag(`A3`, i)
	}
}

func Test_msgq3(t *testing.T) {
	t.Parallel()
	mq := New()

	mun_c := make(chan int, 100)
	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(int); ok {
				mun_c <- v
			}
			return false
		},
	})

	time.Sleep(time.Second)
	for fin_turn := 0; fin_turn < 1000000; fin_turn += 1 {
		// fmt.Printf("\r%d", fin_turn)
		mq.Push_tag(`A1`, fin_turn)
		if fin_turn != <-mun_c {
			t.Fatal(fin_turn)
		}
	}
}

func Test_msgq4(t *testing.T) {
	t.Parallel()
	// mq := New(30)
	mq := New(time.Second, time.Second) //out of list

	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`1`)
			}
			return false
		},
		`A2`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			return false
		},
		`Error`: func(data any) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})
	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			return false
		},
		`Error`: func(data any) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})

	var fin_turn = 0
	time.Sleep(time.Second)
	for fin_turn < 20 {
		go mq.Push_tag(`A1`, `a11`)
		go mq.Push_tag(`A1`, `a11`)
		go mq.Push_tag(`A1`, `a11`)
		// mq.Push_tag(`A4`,`a11`)
		go mq.Push_tag(`A1`, `a11`)
		mq.Push_tag(`A1`, `a11`)
		mq.Push_tag(`A2`, `a11`)
		// mq.Push_tag(`A4`,`a11`)
		// <-mun_c3
		fin_turn += 1
	}
}

func Test_msgq5(t *testing.T) {
	t.Parallel()
	mq := New()

	mun_c1 := make(chan bool, 100)
	mun_c2 := make(chan bool, 100)
	go mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(_ any) bool {
			time.Sleep(time.Second) //will block
			return false
		},
		`A2`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			mun_c2 <- true
			return false
		},
		`Error`: func(data any) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})
	mq.Pull_tag(map[string]func(any) bool{
		`A1`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`1`)
			}
			mun_c1 <- true
			return false
		},
		`A2`: func(data any) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			return false
		},
		`Error`: func(data any) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})

	var fin_turn = 0
	time.Sleep(time.Second)
	for fin_turn < 10 {
		mq.Push_tag(`A1`, `a11`)
		mq.Push_tag(`A2`, `a11`)
		<-mun_c1
		<-mun_c2
		fin_turn += 1
	}
}

func Test_msgq10(t *testing.T) {
	t.Parallel()
	msg := NewType[int]()

	var fc funcCtrl.FlashFunc
	go msg.Pull_tag(map[string]func(int) (disable bool){
		`1`: func(b int) (disable bool) {

			// 当cut时，取消上次录制
			ctx1, done := pctx.WithWait(context.Background(), 1, time.Second*30)
			fc.FlashWithCallback(func() {
				fmt.Println("call done", b)
				done()
				fmt.Println("doned", b)
			})
			fmt.Println("start", b)

			go func() {
				defer fmt.Println("fin", b)

				ctx1, done1 := pctx.WaitCtx(ctx1)
				defer done1()

				cancle := msg.Pull_tag(map[string]func(int) (disable bool){
					`2`: func(i int) (disable bool) {
						if pctx.Done(ctx1) {
							return true
						}
						if b != i {
							t.Logf("should not rev %d when %d has been closed", i, b)
							// t.FailNow()
						}
						return false
					},
				})

				<-ctx1.Done()
				cancle()
			}()
			// if b != 0 {
			// 	t.Fatal()
			// }
			return false
		},
	})
	time.Sleep(time.Second)
	msg.PushLock_tag(`1`, 0)
	msg.PushLock_tag(`2`, 0)
	time.Sleep(time.Second)
	msg.PushLock_tag(`2`, 0)
	msg.PushLock_tag(`1`, 1)
	msg.PushLock_tag(`2`, 1)
	time.Sleep(time.Second)
}

func Test_msgq6(t *testing.T) {
	t.Parallel()
	msg := NewType[int]()
	msg.Pull_tag(map[string]func(int) (disable bool){
		`1`: func(b int) (disable bool) {
			if b != 0 {
				t.Fatal()
			}
			return false
		},
	})
	msg.Push_tag(`1`, 0)
	time.Sleep(time.Second)
}

func Test_msgq7(t *testing.T) {
	t.Parallel()
	var c = make(chan string, 100)
	msg := NewType[int]()
	msg.Pull_tag_async_only(`1`, func(i int) (disable bool) {
		time.Sleep(time.Second)
		c <- "1"
		return i > 10
	})
	msg.Pull_tag_async_only(`1`, func(i int) (disable bool) {
		time.Sleep(time.Second * 2)
		c <- "2"
		return i > 10
	})
	msg.Pull_tag_only(`1`, func(i int) (disable bool) {
		time.Sleep(time.Second * 3)
		c <- "3"
		return i > 10
	})
	msg.Pull_tag_only(`1`, func(i int) (disable bool) {
		time.Sleep(time.Second * 3)
		c <- "4"
		return i > 10
	})
	msg.Push_tag(`1`, 0)
	if i := <-c; i != "1" {
		t.Fatal(i)
	}
	if <-c != "2" {
		t.Fatal()
	}
	if <-c != "3" {
		t.Fatal()
	}
	if <-c != "4" {
		t.Fatal()
	}
}

func Test_msgq8(t *testing.T) {
	t.Parallel()
	msg := NewType[int]()
	msg.Pull_tag_async_only(`1`, func(i int) (disable bool) {
		if i > 4 {
			t.Fatal(i)
		}
		return i > 3
	})
	msg.Pull_tag_only(`1`, func(i int) (disable bool) {
		if i > 6 {
			t.Fatal(i)
		}
		return i > 5
	})
	for i := 0; i < 20; i++ {
		msg.Push_tag(`1`, i)
		time.Sleep(time.Millisecond * 20)
	}
	time.Sleep(time.Second)
}

// func Test_msgq6(t *testing.T) {
// 	mq := New()
// 	go mq.Pull_tag(map[string]func(any) bool{
// 		`A1`: func(data any) bool {
// 			return false
// 		},
// 		`A2`: func(data any) bool {
// 			if v, ok := data.(string); !ok || v != `a11` {
// 				t.Error(`2`)
// 			}
// 			return false
// 		},
// 		`Error`: func(data any) bool {
// 			if data == nil {
// 				t.Error(`out of list`)
// 			}
// 			return false
// 		},
// 	})

// 	var fin_turn = 0
// 	t.Log(`start`)
// 	for fin_turn < 1000 {
// 		time.Sleep(time.Second)
// 		time.Sleep(time.Second)
// 		mq.Push_tag(`A1`, `a11`)
// 		fin_turn += 1
// 		fmt.Print("\r", fin_turn)
// 	}
// 	t.Log(`fin`)
// }
