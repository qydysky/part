package part

import (
	"time"
	"testing"
)

func Test_tmplKV(t *testing.T) {
	s := New_tmplKV()
	s.Set("a",`a`,1)
	if !s.Check("a",`a`) {t.Error(`no match1`)}
	s.Set("a",`b`,-1)
	if !s.Check("a",`b`) {t.Error(`no match2`)}
	time.Sleep(time.Second*time.Duration(1))
	if v,ok := s.Get("a");!ok {
		t.Error(`no TO1`)
	}else if vv,ok := v.(string);!ok{
		t.Error(`no string`)
	}
	if v,ok := s.GetV("a").(string);!ok || v != `a` {t.Error(`no 2`)}
}
