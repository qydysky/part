package part

import (
	_ "embed"
	"testing"
)

var data []byte

func Test_Br(t *testing.T) {
	b, e := InBr(data, 6)
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(len(data))

	c, e := UnBr(b)
	if e != nil {
		t.Error(e)
		return
	}

	for k, v := range c {
		if v != []byte("abc")[k] {
			t.Error(`no match`)
		}
	}
}
