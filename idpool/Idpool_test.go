package part

import (
	"testing"
)

type test struct{}

func Test(t *testing.T) {
	pool := New(func() interface{} {
		return &test{}
	})
	a := pool.Get()
	b := pool.Get()
	t.Log(a.Id, a.Item, pool.Len())
	t.Log(b.Id, b.Item)
	pool.Put(a)
	pool.Put(a)
	t.Log(a.Id, a.Item, pool.Len())
	t.Log(b.Id, b.Item)
	a = pool.Get()
	t.Log(a.Id, a.Item, pool.Len())
	t.Log(b.Id, b.Item)
}
