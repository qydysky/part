package part

import (
	"testing"
)

func Test_signal(t *testing.T) {
	var s *Signal
	t.Log(s.Islive())
	s.Done()
	t.Log(s.Islive())
	s = Init()
	t.Log(s.Islive())
	s.Done()
	t.Log(s.Islive())
}
