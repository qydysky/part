package errors

import (
	"errors"
	"io"
	"testing"
)

func TestXxx(t *testing.T) {
	var err error

	err = New("r0", "a0")

	if !Catch(err, "a0") {
		t.Fail()
	}

	if Catch(err, "a1") {
		t.Fail()
	}

	err = Grow(err, New("r1", "a1"))

	if !Catch(err, "a0") {
		t.Fail()
	}

	if !Catch(err, "a1") {
		t.Fail()
	}
}

func Test2(t *testing.T) {
	e := Join(New("r0", "a0"), New("r1", "a1"))
	t.Log(ErrorFormat(e, ErrSimplifyFunc))
	t.Log(ErrorFormat(e, ErrInLineFunc))
	if ErrorFormat(e, ErrSimplifyFunc) != "r0\nr1\n" {
		t.FailNow()
	}
	if ErrorFormat(e, ErrInLineFunc) != " > r0 > r1" {
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
	if ErrorFormat(e, ErrInLineFunc) != " > EOF > EOF > io: read/write on closed pipe" {
		t.FailNow()
	}
}

func Test3(t *testing.T) {
	e := New("1")
	if e.Error() != "" {
		t.FailNow()
	}
	e1 := e.WithReason("2")
	if e.Error() != "" {
		t.FailNow()
	}
	if e1.Error() != "2" {
		t.FailNow()
	}
}
