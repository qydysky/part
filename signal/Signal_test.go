package part

import (
	"testing"
)

func Test_signal(t *testing.T) {
	var s *Signal
	s.Wait()
	t.Log(s.Islive())
	s.Done()
	t.Log(s.Islive())
	s = Init()
	t.Log(s.Islive())
	s.Done()
	t.Log(s.Islive())
}

func Test_signal2(t *testing.T) {
	s := Init()
	go s.Done()
	s.Wait()
}

func Test_signal3(t *testing.T) {
	var s *Signal
	go func() {
		if s != nil {
			s.Islive()
		}
	}()
	s = Init()
}

func BenchmarkXxx(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := Init()
		go s.Done()
		s.Wait()
	}
}
