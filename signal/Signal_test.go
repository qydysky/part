package part

import (
	"testing"
)

func Test_signal(t *testing.T) {
	var s *Signal = Init()
	if !s.Islive() {
		t.Fatal()
	}
	s.Done()
	if s.Islive() {
		t.Fatal()
	}
}

func Test_signal2(t *testing.T) {
	s := Init()
	go s.Done()
	s.Wait()
}

func BenchmarkXxx(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := Init()
		go s.Done()
		s.Wait()
	}
}
