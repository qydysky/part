package part

import (
	"testing"
)

func Test(t *testing.T){
	pool := New()
	a := pool.Get()
	b := pool.Get()
	t.Log(a.Id,a.item,pool.Len())
	t.Log(b.Id,b.item)
	pool.Put(a)
	t.Log(a.Id,a.item,pool.Len())
	t.Log(b.Id,b.item)
	a = pool.Get()
	t.Log(a.Id,a.item,pool.Len())
	t.Log(b.Id,b.item)
}
