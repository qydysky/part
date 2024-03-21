package bools

import "testing"

func TestMain(t *testing.T) {
	s := New("{1}", map[string]func() bool{
		"1": True,
		"2": False,
	})
	if ok, e := s.SetRule("({2}&!{2})").Check(); e != nil || ok {
		t.Fatal(e)
	}
	if ok, e := s.Check(); e != nil || !ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("!{1}").Check(); e != nil || ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("!!{1}").Check(); e != nil || !ok {
		t.Fatal()
	}
	if _, e := s.SetRule("{1}{2}").Check(); e != ErrNoAct {
		t.Fatal()
	}
	if ok, e := s.SetRule("{1}|{2}").Check(); e != nil || !ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("{1}&{2}").Check(); e != nil || ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("{1}&!{2}").Check(); e != nil || !ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("!{1}&!{2}&!{2}").Check(); e != nil || ok {
		t.Fatal(e)
	}
	if ok, e := s.SetRule("!{1}&!{2}&!{2}").Check(); e != nil || ok {
		t.Fatal(e)
	}
	if ok, e := s.SetRule("({1}&{2})").Check(); e != nil || ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("{1}&({1}&{2})").Check(); e != nil || ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("{1}&!({1}&{2})").Check(); e != nil || !ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("!{1}&({1}&!{2})").Check(); e != nil || ok {
		t.Fatal()
	}
	if ok, e := s.SetRule("{2}&({1}&{1})").Check(); e != nil || ok {
		t.Fatal()
	}
}
