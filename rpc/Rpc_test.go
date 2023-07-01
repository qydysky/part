package part

import (
	"errors"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	pob := Pob{Host: "127.0.0.1:10902", Path: "/123"}
	if shutdown, e := pob.Server(func(i, o *Gob) error {
		if iv, ok := i.Data.(int); !ok {
			return errors.New("iv")
		} else {
			switch i.Key {
			case "+":
				o.Data = iv + 1
			case "-":
				o.Data = iv - 1
			default:
				return errors.New("no key")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	} else {
		defer shutdown()
	}

	time.Sleep(time.Second)

	if c, e := pob.Client(); e != nil {
		t.Fatal(e)
	} else {
		var gob = Gob{"+", 9}
		if e := c.Call(&gob); e != nil {
			t.Fatal(e)
		} else if gob.Key != "+" {
			t.Fatal()
		} else if i, ok := gob.Data.(int); !ok || i != 10 {
			t.Fatal()
		}
		c.Close()
	}

	if c, e := pob.Client(); e != nil {
		t.Fatal(e)
	} else {
		var gob = Gob{"-", 9}
		if e := c.Call(&gob); e != nil {
			t.Fatal(e)
		} else if gob.Key != "-" {
			t.Fatal()
		} else if i, ok := gob.Data.(int); !ok || i != 8 {
			t.Fatal()
		}
		c.Close()
	}
}
