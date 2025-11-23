package part

import (
	"bytes"
	"math"
	"sync"
	"testing"
	"time"
	"unsafe"
)

type a struct {
	d []byte
	v bool
}

func Benchmark3(b *testing.B) {
	p := New(PoolFunc[[]int]{}, -1)
	for b.Loop() {
		m := p.Get()
		*m = append((*m)[:0], 0)
		p.Put(m)
	}
}

func Benchmark1(b *testing.B) {
	p := sync.Pool{
		New: func() any {
			return &[]int{}
		},
	}
	for b.Loop() {
		m := p.Get().(*[]int)
		*m = append((*m)[:0], 0)
		p.Put(m)
	}
}

func Test2(t *testing.T) {
	p := New(PoolFunc[int]{}, 1)
	t.Log(p.pf.InUse == nil)
}

func Test1(t *testing.T) {
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

	var b = New(PoolFunc[a]{newf, validf, reusef, poolf}, 10)

	for i := 0; i < 10; i++ {
		b.Get()
	}
	time.Sleep(time.Millisecond * 1100)
	if math.Abs(b.State().GetPerSec-7.5) > 2.5 {
		t.Fatal(b.State().GetPerSec)
	}
}

func TestXxx(t *testing.T) {
	var b = New(PoolFunc[a]{
		New: func() *a {
			return &a{v: true}
		},
		InUse: func(t *a) bool {
			return t.v
		},
		Reuse: func(t *a) *a {
			t.d = t.d[:0]
			t.v = true
			return t
		}, Pool: func(t *a) *a {
			return t
		}}, 10)

	var c1 = b.Get()
	var c1p = uintptr(unsafe.Pointer(c1))
	c1.d = append(c1.d, []byte("1")...)

	var c2 = b.Get()
	var c2p = uintptr(unsafe.Pointer(c2))
	c2.d = append(c2.d, []byte("2")...)

	if c1p == c2p {
		t.Fatal()
	} else if bytes.Equal(c1.d, c2.d) {
		t.Fatal()
	} else if b.State().Inuse != 0 {
		t.Fatal()
	} else if b.State().Sum != 0 {
		t.Fatal()
	}

	c1.v = false

	var c3 = b.Get()
	var c3p = uintptr(unsafe.Pointer(c3))

	if c1p == c3p || len(c1.d) == 0 || b.State().Inuse != 0 || b.State().Sum != 0 {
		t.Fatal()
	}

	b.Put(c1)

	if len(c1.d) == 0 || b.State().Inuse != 0 || b.State().Sum != 1 {
		t.Fatal(len(c1.d) != 0, b.State().Inuse != 0, b.State().Sum != 1)
	}
}
