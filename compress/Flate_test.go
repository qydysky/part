package part

import (
	"testing"
)

func Test_Flate(t *testing.T) {
	s := []byte(`abc`)
	t.Log(string(s))

	b,e := InFlate(s, -1)
	if e != nil {t.Error(e);return}
	t.Log(string(b))

	c,e := UnFlate(b)
	if e != nil {t.Error(e);return}
	t.Log(string(c))
	
	for k,v := range c {
		if v != []byte("abc")[k] {t.Error(`no match`)}
	}
}
