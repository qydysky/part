package part

import (
	"container/list"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Msgq struct {
	funcs          *list.List
	someNeedRemove atomic.Int32
	lock           sync.RWMutex
}

type FuncMap map[string]func(any) (disable bool)

func New() *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	return m
}

func (m *Msgq) Register(f func(any) (disable bool)) {
	m.lock.Lock()
	m.funcs.PushBack(f)
	m.lock.Unlock()
}

func (m *Msgq) Push(msg any) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	var removes []*list.Element

	m.lock.RLock()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			m.someNeedRemove.Add(1)
			removes = append(removes, el)
		}
	}
	m.lock.RUnlock()

	if len(removes) != 0 {
		m.lock.Lock()
		m.someNeedRemove.Add(-int32(len(removes)))
		for i := 0; i < len(removes); i++ {
			m.funcs.Remove(removes[i])
		}
		m.lock.Unlock()
		removes = nil
	}
}

type Msgq_tag_data struct {
	Tag  string
	Data any
}

func (m *Msgq) Push_tag(Tag string, Data any) {
	m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) Pull_tag(func_map map[string]func(any) (disable bool)) {
	m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data)
			}
		}
		return false
	})
}

type MsgType[T any] struct {
	m *Msgq
}

func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	if m.m == nil {
		m.m = New()
	}
	m.m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) {
	if m.m == nil {
		m.m = New()
	}
	m.m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data.(T))
			}
		}
		return false
	})
}
