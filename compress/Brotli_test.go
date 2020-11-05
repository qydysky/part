package part

import (
	"testing"
)

func Test_Br(t *testing.T) {
	s := []byte(`abc`)
	t.Log(string(s))

	b,e := InBr(s, 6)
	if e != nil {t.Error(e);return}
	t.Log(string(b))

	c,e := UnBr(b)
	if e != nil {t.Error(e);return}
	t.Log(string(c))
	
	for k,v := range c {
		if v != []byte("abc")[k] {t.Error(`no match`)}
	}
}
