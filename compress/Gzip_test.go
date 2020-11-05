package part

import (
	"testing"
)

func Test_Gzip(t *testing.T) {
	s := []byte(`abc`)
	t.Log(string(s))

	b,e := InGzip(s, -1)
	if e != nil {t.Error(e);return}
	t.Log(string(b))

	c,e := UnGzip(b)
	if e != nil {t.Error(e);return}
	t.Log(string(c))
	
	for k,v := range c {
		if v != []byte("abc")[k] {t.Error(`no match`)}
	}
}
