package part

import (
	"container/list"
	"sync"
)

type Msgq struct {
	funcs *list.List
	sync.RWMutex
}

func New() *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	return m
}

func (m *Msgq) Register(f func(any) (disable bool)) {
	m.Lock()
	m.funcs.PushBack(f)
	m.Unlock()
}

func (m *Msgq) Push(msg any) {
	m.RLock()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			m.funcs.Remove(el)
		}
	}
	m.RUnlock()
}

type Msgq_tag_data struct {
	Tag  string
	Data interface{}
}

func (m *Msgq) Push_tag(Tag string, Data interface{}) {
	m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) Pull_tag(func_map map[string]func(any) (disable bool)) {
	m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); !ok {
			if f, ok := func_map[`Error`]; ok {
				return f(d.Data)
			}
		} else {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data)
			}
		}
		return false
	})
}
