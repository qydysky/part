package part

import (
	"time"
	"testing"
)

func Test_tmplV(t *testing.T) {
	s := New_tmplV(1e6, 1)
	v := s.Set("a")
	if o,p := s.Buf();p != 1 || o - time.Now().Unix() > 1{return}
	if ok,k := s.Get(v);!ok || k != "a" {return}
	if !s.Check(v, "a") {return}
}
