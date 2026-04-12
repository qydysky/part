package v2

import (
	"errors"
	"io"
	"testing"
)

var bus, actM = Action[struct {
	A Error `err:"a"`
	B *Error
}](`bus`)

func Test1(t *testing.T) {
	t.Log(actM.Info())
	t.Log(ErrorFormat(bus.A, ErrActionInLineFunc))
	if bus.A.Error() != `a` {
		t.Fatal()
	}
	if bus.B.Error() != `B` {
		t.Fatal()
	}
	a := error(bus.A)
	if !actM.InAction(a) {
		t.Fatal()
	}
	if !errors.Is(a, bus.A) {
		t.Fatal()
	}
	if actM.InAction(io.EOF) {
		t.Fatal()
	}
	b := errors.Join(io.EOF, bus.A, io.EOF)
	t.Log(ErrorFormat(b, ErrActionInLineFunc))
	if !errors.Is(b, bus.A) {
		t.Fatal()
	}
	if !actM.InAction(b) {
		t.Fatal()
	}
	b = errors.Join(b, io.EOF)
	if !actM.InAction(b) {
		t.Fatal()
	}
}

func Test2(t *testing.T) {
	bus, actM := Action[struct {
		B Error
	}](`bus`)
	bus2, actM2 := Action[struct {
		C Error
	}](`bus2`)
	b := errors.Join(bus.B, bus2.C)
	if !actM2.InAction(b) {
		t.Fatal()
	}
	if !actM.InAction(b) {
		t.Fatal()
	}
}

func Test3(t *testing.T) {
	bus, actM := Action[struct {
		B Error
	}](`bus`)
	a := bus.B.Wrap(io.EOF)
	t.Log(ErrorFormat(a, ErrActionInLineFunc))
	if !actM.InAction(bus.B) {
		t.Fatal()
	}
	if !actM.InAction(a) {
		t.Fatal()
	}
	if !errors.Is(a, bus.B) {
		t.Fatal()
	}
	if !errors.Is(a, io.EOF) {
		t.Fatal()
	}
	if !errors.Is(bus.B, io.EOF) {
		t.Fatal()
	}
	b := bus.B
	if !errors.Is(b, bus.B) {
		t.Fatal()
	}
	c := errors.Join(io.EOF, bus.B, io.EOF)
	if !errors.Is(c, bus.B) {
		t.Fatal()
	}
	if !actM.InAction(c) {
		t.Fatal()
	}
}

func Benchmark1(b *testing.B) {
	a1, a1M := Action[struct {
		A Error
	}](`a1`)
	err := errors.Join(a1.A, io.EOF)
	for b.Loop() {
		if !a1M.InAction(err) {
			b.Fatal()
		}
	}
}
