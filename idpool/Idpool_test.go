package part

import (
	"testing"
)

type test struct{}

func Test(t *testing.T) {
	pool := New(func() *test {
		return new(test)
	})
	a := pool.Get()
	b := pool.Get()
	if pool.InUse() != 2 {
		t.Fatal()
	}
	bid := b.Id
	pool.Put(b)
	if pool.InUse() != 1 {
		t.Fatal()
	}
	a = pool.Get()
	if bid != a.Id {
		t.Fatal()
	}
}
