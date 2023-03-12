package part

import (
	_ "net/http/pprof"
	"testing"
	"time"
)

type test_item struct {
	data string
}

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
// 			data interface{}
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
// 			data interface{}
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
// 			data interface{}
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

func Test_msgq2(t *testing.T) {
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
	mq := New()

	mun_c := make(chan int, 100)
	mq.Pull_tag(map[string]func(interface{}) bool{
		`A1`: func(data interface{}) bool {
			if v, ok := data.(int); ok {
				mun_c <- v
			}
			return false
		},
	})

	time.Sleep(time.Second)
	for fin_turn := 0; fin_turn < 1000000; fin_turn += 1 {
		mq.Push_tag(`A1`, fin_turn)
		if fin_turn != <-mun_c {
			t.Fatal(fin_turn)
		}
	}
}

func Test_msgq4(t *testing.T) {
	// mq := New(30)
	mq := New() //out of list

	mun_c1 := make(chan bool, 100)
	mun_c2 := make(chan bool, 100)
	mun_c3 := make(chan bool, 100)
	mq.Pull_tag(map[string]func(interface{}) bool{
		`A1`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`1`)
			}
			mun_c1 <- true
			return false
		},
		`A2`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			mun_c2 <- true
			return false
		},
		`Error`: func(data interface{}) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})
	mq.Pull_tag(map[string]func(interface{}) bool{
		`A1`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			mun_c3 <- true
			return false
		},
		`Error`: func(data interface{}) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})

	var fin_turn = 0
	time.Sleep(time.Second)
	for fin_turn < 5 {
		go mq.Push_tag(`A1`, `a11`)
		go mq.Push_tag(`A1`, `a11`)
		go mq.Push_tag(`A1`, `a11`)
		// mq.Push_tag(`A4`,`a11`)
		go mq.Push_tag(`A1`, `a11`)
		mq.Push_tag(`A1`, `a11`)
		mq.Push_tag(`A2`, `a11`)
		// mq.Push_tag(`A4`,`a11`)
		<-mun_c2
		<-mun_c1
		// <-mun_c3
		fin_turn += 1
	}
}

func Test_msgq5(t *testing.T) {
	mq := New()

	mun_c1 := make(chan bool, 100)
	mun_c2 := make(chan bool, 100)
	go mq.Pull_tag(map[string]func(interface{}) bool{
		`A1`: func(data interface{}) bool {
			time.Sleep(time.Second) //will block
			return false
		},
		`A2`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			mun_c2 <- true
			return false
		},
		`Error`: func(data interface{}) bool {
			if data == nil {
				t.Error(`out of list`)
			}
			return false
		},
	})
	mq.Pull_tag(map[string]func(interface{}) bool{
		`A1`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`1`)
			}
			mun_c1 <- true
			return false
		},
		`A2`: func(data interface{}) bool {
			if v, ok := data.(string); !ok || v != `a11` {
				t.Error(`2`)
			}
			return false
		},
		`Error`: func(data interface{}) bool {
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

func Test_msgq6(t *testing.T) {
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
	if <-c != "1" {
		t.Fatal()
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
// 	go mq.Pull_tag(map[string]func(interface{}) bool{
// 		`A1`: func(data interface{}) bool {
// 			return false
// 		},
// 		`A2`: func(data interface{}) bool {
// 			if v, ok := data.(string); !ok || v != `a11` {
// 				t.Error(`2`)
// 			}
// 			return false
// 		},
// 		`Error`: func(data interface{}) bool {
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
