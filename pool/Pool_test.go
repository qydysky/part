package part

import (
	"bytes"
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
	var c1p = uintptr(unsafe.Pointer(c1))
	c1.d = append(c1.d, []byte("1")...)

	var c2 = b.Get()
	var c2p = uintptr(unsafe.Pointer(c2))
	c2.d = append(c2.d, []byte("2")...)

	if c1p == c2p || bytes.Equal(c1.d, c2.d) || b.PoolInUse() != 0 || b.PoolSum() != 0 {
		t.Fatal()
	}

	b.Put(c1)
	c1.v = false
	var c3 = b.Get()
	var c3p = uintptr(unsafe.Pointer(c3))

	if c1p != c3p || len(c1.d) != 0 || b.PoolInUse() != 1 || b.PoolSum() != 1 {
		t.Fatal()
	}
}
