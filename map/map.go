package part

import (
	"sync"
)

type Map struct{
	m sync.Map
	New func()interface{}
}

func (m *Map) Get(key interface{})interface{}{
	if val,ok := m.Load(key);ok{return val}
	return m.Set(key, m.New())
}

func (m *Map) Del(key interface{}){
	m.Delete(key)
}

func (m *Map) Set(key interface{},val interface{})interface{}{
	m.Store(key, val)
	return val
}