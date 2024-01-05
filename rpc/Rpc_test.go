package part

import (
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
	pob := NewServer("127.0.0.1:10902")
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

	if e := Call("127.0.0.1:10902", "/123", &i, &out); e != nil {
		t.Fatal(e)
	}

	if out.Data.Data != 10 {
		t.FailNow()
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
