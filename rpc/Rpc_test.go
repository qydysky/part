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

	if e := Call(&i, &out, "127.0.0.1:10902", "/123"); e != nil {
		t.Fatal(e)
	}

	if out.Data.Data != 10 {
		t.FailNow()
	}
}
