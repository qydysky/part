package part

import (
	"testing"
	"time"
)

func TestMain(t *testing.T) {
}

func BenchmarkRlock(b *testing.B) {
	var lock1 RWMutex
	for i := 0; i < b.N; i++ {
		lock1.RLock(time.Second, time.Second)()
	}
}
