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

func Test1(t *testing.T) {
	e := Join(io.EOF, io.ErrClosedPipe)
	e = Join(io.EOF, e)
	if !errors.Is(e, io.ErrClosedPipe) {
		t.FailNow()
	}
	if ErrorFormat(e, ErrSimplifyFunc) != "EOF\nEOF\nio: read/write o...\n" {
		t.FailNow()
	}
}
