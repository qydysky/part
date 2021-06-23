package part

import (
	"sync"
)

type Map struct{
	m sync.Map
	New func()interface{}
}

func (t *Map) Get(key interface{})interface{}{
	if val,ok := t.m.Load(key);ok{return val}
	return t.Set(key, t.New())
}

func (t *Map) Del(key interface{}){
	t.m.Delete(key)
}

func (t *Map) Set(key interface{},val interface{})interface{}{
	t.m.Store(key, val)
	return val
}