package part

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type test1 struct {
	Data test1_1
}
type test1_1 struct {
	Data int
}

type test2 struct {
	Data test2_1
}
type test2_1 struct {
	Data int
}

func TestMain(t *testing.T) {
	pob := NewServer("127.0.0.1:10904")
	defer pob.Shutdown()
	if e := Register(pob, "/123", func(i *int, o *test1) error {
		*i += 1
		o.Data.Data = *i
		return nil
	}); e != nil {
		t.Fatal(e)
	}

	time.Sleep(time.Second)

	var i int = 9
	var out test2

	if e := Call("127.0.0.1:10904", "/123", &i, &out); e != nil {
		t.Fatal(e)
	}

	if out.Data.Data != 10 {
		t.FailNow()
	}
}

func TestMain2(t *testing.T) {
	pob := NewServer("127.0.0.1:10903")
	defer pob.Shutdown()
	if e := Register(pob, "/add", func(i *int, o *int) error {
		*o = *i + 1
		return nil
	}); e != nil {
		t.Fatal(e)
	}

	var in int = 1
	var out int
	if e := Call("127.0.0.1:10903", "/add", &in, &out); e != nil {
		t.Fatal(e)
	}

	if out != 2 {
		t.FailNow()
	}
}

func TestMain3(t *testing.T) {
	pob := NewServer("127.0.0.1:10903")
	defer pob.Shutdown()

	var fileLock sync.Map
	if e := Register(pob, "/lock", func(i *string, o *bool) error {
		if _, ok := fileLock.LoadOrStore(*i, nil); !ok {
			*o = true
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e := Register(pob, "/unlock", func(i *string, o *bool) error {
		_, *o = fileLock.LoadAndDelete(*i)
		return nil
	}); e != nil {
		t.Fatal(e)
	}

	lock := func(path string) (ok bool, e error) {
		e = Call("127.0.0.1:10903", "/lock", &path, &ok)
		return
	}

	unlock := func(path string) (ok bool, e error) {
		e = Call("127.0.0.1:10903", "/unlock", &path, &ok)
		return
	}

	if ok, e := lock("./a.exe"); e != nil {
		t.Fatal(e)
	} else if !ok {
		t.Fatal()
	}

	if ok, e := lock("./a.exe"); e != nil {
		t.Fatal(e)
	} else if ok {
		t.Fatal()
	}

	if ok, e := unlock("./a.exe"); e != nil {
		t.Fatal(e)
	} else if !ok {
		t.Fatal()
	}

	if ok, e := unlock("./a.exe"); e != nil {
		t.Fatal(e)
	} else if ok {
		t.Fatal()
	}
}

func TestMain4(t *testing.T) {
	pob := NewServer("127.0.0.1:10903")
	defer pob.Shutdown()
	if e := Register(pob, "/add", func(i *int, o *int) error {
		*o = *i + 1
		time.Sleep(time.Millisecond * 100)
		return nil
	}); e != nil {
		t.Fatal(e)
	}

	caller := CallReuse[int, int]("127.0.0.1:10903", "/add", 10)

	call := func() error {
		var in int = 1
		var out int
		if e := caller(&in, &out); e != nil {
			return e
		} else if out != 2 {
			return errors.New("")
		}
		return nil
	}

	s := time.Now()
	go func() {
		if e := call(); e != nil {
			t.Fatal(e)
		}
		fmt.Println(time.Since(s))
	}()
	go func() {
		if e := call(); e != nil {
			t.Fatal(e)
		}
		fmt.Println(time.Since(s))
	}()
	time.Sleep(time.Millisecond * 500)
}

func TestMain5(t *testing.T) {
	var i int = 10222
	var o int
	g := new(Gob)
	g.encode(&i)
	g.decode(&o)
	if i != o {
		t.Fatal()
	}
}

func Benchmark1(b *testing.B) {
	pob := NewServer("127.0.0.1:10903")
	defer pob.Shutdown()
	if e := Register(pob, "/add", func(i *int, o *int) error {
		*o = *i + 1
		return nil
	}); e != nil {
		b.Fatal(e)
	}

	caller := CallReuse[int, int]("127.0.0.1:10903", "/add", 500)

	for b.Loop() {
		var in int = 1
		var out int
		if e := caller(&in, &out); e != nil || out != 2 {
			b.Fatal(e)
		}
	}
}

// func TestMain2(t *testing.T) {

// 	pob := NewServer("127.0.0.1:10902")
// 	defer pob.Shutdown()
// 	if e := Register(pob, "/123", func(i *int, o *struct{}) error {
// 		if *i == 9 || *i == 10 {
// 			t.FailNow()
// 		}
// 		return errors.New("")
// 	}); e != nil {
// 		t.Fatal(e)
// 	}

// 	pob2 := NewServer("127.0.0.1:10903")
// 	defer pob2.Shutdown()
// 	if e := Register(pob2, "/123", func(i *int, o *struct{}) error {

// 		if e := RegisterSerReg("127.0.0.1:10992", "/", RegisterSerHost{"del", "127.0.0.1:10903", "/123"}); e != nil {
// 			t.Fatal(e)
// 		}

// 		if *i == 9 {
// 			t.FailNow()
// 		}
// 		return errors.New("")
// 	}); e != nil {
// 		t.Fatal(e)
// 	}

// 	regs, e := NewRegisterSer("127.0.0.1:10992", "/")
// 	if e != nil {
// 		t.Fatal(e)
// 	} else {
// 		defer regs.Shutdown()
// 	}

// 	if e := RegisterSerReg("127.0.0.1:10992", "/", RegisterSerHost{"add", "127.0.0.1:10902", "/123"}); e != nil {
// 		t.Fatal(e)
// 	}
// 	if e := RegisterSerReg("127.0.0.1:10992", "/", RegisterSerHost{"add", "127.0.0.1:10903", "/123"}); e != nil {
// 		t.Fatal(e)
// 	}

// 	var i int = 9

// 	RegisterSerCall(regs, &i, "/123")

// 	i++

// 	RegisterSerCall(regs, &i, "/123")
// }
