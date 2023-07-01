package part

import (
	"errors"
	"log"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	pob := Pob{Host: "127.0.0.1:10902", Path: "/123"}
	if shutdown, e := pob.Server(func(i, o *Gob) error {
		switch i.Key {
		case "+":
			var ivv int
			if e := i.Decode(&ivv); e != nil {
				log.Fatal("d", e)
			}
			ivv += 1
			if e := o.Encode(ivv); e != nil {
				log.Fatal("e", e)
			}
		default:
			return errors.New("no key")
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
		var gob = NewGob("+")

		var i int = 9
		if e := gob.Encode(&i); e != nil {
			t.Fatal(e)
		}
		if e := gob.Decode(&i); e != nil {
			t.Fatal(e)
		}
		if e := c.Call(gob); e != nil {
			t.Fatal(e)
		} else if gob.Key != "+" {
			t.Fatal()
		} else {
			if e := gob.Decode(&i); e != nil {
				t.Fatal(e)
			}
			if i != 10 {
				t.Fatal()
			}
		}
		c.Close()
	}

}
