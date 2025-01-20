package errors

import (
	"errors"
	"io"
	"testing"
)

var a0 = Action("a0")
var a1 = Action("a1")
var a11 = a1.Append("1")

func TestXxx(t *testing.T) {
	var err error

	err = New("r0", a0)

	if !Catch(err, a0) {
		t.Fail()
	}

	if Catch(err, "r0") {
		t.Fail()
	}

	err = Append(New("r1", a11), err)

	if !Catch(err, a11) {
		t.Fail()
	}

	if !Catch(err, a0) {
		t.Fail()
	}
}

func TestXxx2(t *testing.T) {
	err := Append(New("r1", a1), io.EOF)
	if !Catch(err, a1) {
		t.Fatal()
	}
	t.Log(err.Error())
}

func Test2(t *testing.T) {
	e := Join(New("r0", a0), New("r1", a1))
	t.Log(ErrorFormat(e))
	t.Log(ErrorFormat(e, ErrSimplifyFunc))
	t.Log(ErrorFormat(e, ErrInLineFunc))
	if ErrorFormat(e, ErrSimplifyFunc) != "r0\nr1\n" {
		t.FailNow()
	}
	if ErrorFormat(e, ErrInLineFunc) != "> r0 > r1 " {
		t.FailNow()
	}
}

func Test1(t *testing.T) {
	e := Join(io.EOF, io.ErrClosedPipe)
	e = Join(io.EOF, e)
	if !errors.Is(e, io.ErrClosedPipe) {
		t.FailNow()
	}
	if ErrorFormat(e, ErrSimplifyFunc) != "EOF\nEOF\nio: read/write on closed pipe\n" {
		t.FailNow()
	}
	if ErrorFormat(e, ErrInLineFunc) != "> EOF > EOF > io: read/write on closed pipe " {
		t.FailNow()
	}
}

func Test3(t *testing.T) {
	e := New("1")
	if e.Error() != "1" {
		t.Fatal()
	}
	e1 := e.WithReason("2")
	if e.Error() != "1" {
		t.Fatal()
	}
	if e1.Error() != "2" {
		t.Fatal()
	}
}
