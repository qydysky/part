package errors

import "testing"

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
