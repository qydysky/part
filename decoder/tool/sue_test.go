package tool

import (
	"testing"
)

func Test_ue(t *testing.T) {
	if r, m := ue(0b01101100); r != 2 || m != 3 {
		t.Fatal()
	}
}

func Test_se(t *testing.T) {
	if r, m := se(0b00101000); r != 2 || m != 5 {
		t.Fatal()
	}
	if r, m := se(0b00100000); r != -2 || m != 5 {
		t.Fatal()
	}
}
