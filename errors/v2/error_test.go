package v2

import (
	"errors"
	"io"
	"testing"
)

var bus = Action[struct {
	actM Method
	A    Error `err:"a"`
	B    Error
}](`bus`)

func Test1(t *testing.T) {
	t.Log(bus.actM.Info())
	t.Log(ErrorFormat(bus.A, ErrActionInLineFunc))
	if bus.A.Error() != `a` {
		t.Fatal(bus.A.Error())
	}
	if bus.B.Error() != `B` {
		t.Fatal()
	}
	a := error(bus.A)
	if !bus.actM.InAction(a) {
		t.Fatal()
	}
	if !errors.Is(a, bus.A) {
		t.Fatal()
	}
	if bus.actM.InAction(io.EOF) {
		t.Fatal()
	}
	b := errors.Join(io.EOF, bus.A, io.EOF)
	t.Log(ErrorFormat(b, ErrActionInLineFunc))
	if !errors.Is(b, bus.A) {
		t.Fatal()
	}
	if !bus.actM.InAction(b) {
		t.Fatal()
	}
	b = errors.Join(b, io.EOF)
	if !bus.actM.InAction(b) {
		t.Fatal()
	}
}

func Test2(t *testing.T) {
	bus := Action[struct {
		actM Method
		B    Error
	}](`bus`)
	bus2 := Action[struct {
		actM2 Method
		C     Error
	}](`bus2`)
	b := errors.Join(bus.B, bus2.C)
	if !bus2.actM2.InAction(b) {
		t.Fatal()
	}
	if !bus.actM.InAction(b) {
		t.Fatal()
	}
}

func Test3(t *testing.T) {
	bus := Action[struct {
		actM Method
		B    Error
	}](`bus`)
	a := Join(bus.B.Raw("we"), io.EOF)
	t.Log(ErrorFormat(a, ErrActionInLineFunc))
	if !bus.actM.InAction(bus.B) {
		t.Fatal()
	}
	if !bus.actM.InAction(a) {
		t.Fatal()
	}
	if !errors.Is(a, bus.B) {
		t.Fatal()
	}
	if !errors.Is(a, io.EOF) {
		t.Fatal()
	}
	if !errors.Is(a, io.EOF) {
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
	if !bus.actM.InAction(c) {
		t.Fatal()
	}
}

func Test5(t *testing.T) {
	if testing.Benchmark(Benchmark2).AllocsPerOp() != 0 {
		t.Fatal()
	}
}

func Benchmark2(b *testing.B) {
	a := Action[struct {
		A Error
	}](``)
	c := a.A.Raw(`123`)
	for b.Loop() {
		c.Error()
	}
}

func Benchmark1(b *testing.B) {
	a1 := Action[struct {
		actM Method
		A    Error
	}](`a1`)
	err := errors.Join(a1.A, io.EOF)
	for b.Loop() {
		if !a1.actM.InAction(err) {
			b.Fatal()
		}
	}
}

func Test4(t *testing.T) {
	bus := Action[struct {
		actM Method
		B    Error
	}](`bus`)
	b := bus.B.Raw("1")
	_ = bus.B.Raw("2")
	if b.Error() != "B:1" {
		t.Fatal(b)
	}
	if !errors.Is(b, bus.B) {
		t.Fatal()
	}
	var a = Error{
		fieldName: bus.B.fieldName,
		raw:       bus.B.raw,
		point:     bus.B.point,
		actRaw:    bus.B.actRaw,
		actPoint:  bus.B.actPoint,
	}
	if !bus.actM.InAction(a) {
		t.Fatal()
	}
	if !errors.Is(a, bus.B) {
		t.Fatal()
	}
}
