package part

import (
	"testing"
	"unsafe"
)

type a struct {
	d []byte
	v bool
}

func TestXxx(t *testing.T) {
	var newf = func() *a {
		return &a{v: true}
	}

	var validf = func(t *a) bool {
		return t.v
	}

	var reusef = func(t *a) *a {
		t.d = t.d[:0]
		t.v = true
		return t
	}

	var poolf = func(t *a) *a {
		return t
	}

	var b = New(newf, validf, reusef, poolf, 10)

	var c1 = b.Get()
	c1.d = append(c1.d, []byte("1")...)

	t.Log(unsafe.Pointer(c1), c1)

	var c2 = b.Get()
	c2.d = append(c2.d, []byte("2")...)

	t.Log(unsafe.Pointer(c2), c2)

	b.Put(c1)

	t.Log(unsafe.Pointer(c1), c1)

	c1.v = false

	t.Log(unsafe.Pointer(c1), c1)

	var c3 = b.Get()

	t.Log(unsafe.Pointer(c3), c3)

}
