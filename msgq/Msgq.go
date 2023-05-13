package part

import (
	"container/list"
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	signal "github.com/qydysky/part/signal"
	sync "github.com/qydysky/part/sync"
)

type Msgq struct {
	to             []time.Duration
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

func NewTo(to ...time.Duration) *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	if len(to) > 0 {
		m.to = append(m.to, to[0])
	} else {
		m.to = append(m.to, time.Second*30)
	}
	return m
}

func (m *Msgq) Register(f func(any) (disable bool)) {
	ul := m.lock.Lock(m.to...)
	m.funcs.PushBack(f)
	ul()
}

func (m *Msgq) Register_front(f func(any) (disable bool)) {
	ul := m.lock.Lock(m.to...)
	m.funcs.PushFront(f)
	ul()
}

func (m *Msgq) Push(msg any) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	var removes []*list.Element

	ul := m.lock.RLock(m.to...)
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			m.someNeedRemove.Add(1)
			removes = append(removes, el)
		}
	}
	ul()

	if len(removes) != 0 {
		ul := m.lock.Lock(m.to...)
		m.someNeedRemove.Add(-int32(len(removes)))
		for i := 0; i < len(removes); i++ {
			m.funcs.Remove(removes[i])
		}
		ul()
	}
}

func (m *Msgq) PushLock(msg any) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	ul := m.lock.Lock(m.to...)
	defer ul()

	var removes []*list.Element

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			m.someNeedRemove.Add(1)
			removes = append(removes, el)
		}
	}

	if len(removes) != 0 {
		m.someNeedRemove.Add(-int32(len(removes)))
		for i := 0; i < len(removes); i++ {
			m.funcs.Remove(removes[i])
		}
	}
}

func (m *Msgq) ClearAll() {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	ul := m.lock.Lock(m.to...)
	defer ul()

	var removes []*list.Element

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		m.someNeedRemove.Add(1)
		removes = append(removes, el)
	}

	if len(removes) != 0 {
		m.someNeedRemove.Add(-int32(len(removes)))
		for i := 0; i < len(removes); i++ {
			m.funcs.Remove(removes[i])
		}
	}
}

type Msgq_tag_data struct {
	Tag  string
	Data any
}

func (m *Msgq) Push_tag(Tag string, Data any) {
	defer func() {
		if e := recover(); e != nil {
			panic(fmt.Sprintf("Push_tag(%s,%v) > %v", Tag, Data, e))
		}
	}()
	m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) PushLock_tag(Tag string, Data any) {
	defer func() {
		if e := recover(); e != nil {
			panic(fmt.Sprintf("PushLock_tag(%s,%v) > %v", Tag, Data, e))
		}
	}()
	m.PushLock(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) Pull_tag_chan(key string, size int, ctx context.Context) <-chan any {
	var ch = make(chan any, size)
	m.Register(func(data any) bool {
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			select {
			case <-ctx.Done():
				close(ch)
				return true
			default:
				if len(ch) == size {
					<-ch
				}
				ch <- d.Data
				return false
			}
		}
		return false
	})
	return ch
}

func (m *Msgq) Pull_tag_only(key string, f func(any) (disable bool)) {
	m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			return f(d.Data)
		}
		return false
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

func (m *Msgq) Pull_tag_async_only(key string, f func(any) (disable bool)) {
	var disable = signal.Init()

	m.Register_front(func(data any) bool {
		if !disable.Islive() {
			return true
		}
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			go func() {
				if f(d.Data) {
					disable.Done()
				}
			}()
		}
		return false
	})
}

func (m *Msgq) Pull_tag_async(func_map map[string]func(any) (disable bool)) {
	var disable = signal.Init()

	m.Register_front(func(data any) bool {
		if !disable.Islive() {
			return true
		}
		if d, ok := data.(Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					if f(d.Data) {
						disable.Done()
					}
				}()
			}
		}
		return false
	})
}

type MsgType[T any] struct {
	m *Msgq
}

func NewType[T any]() *MsgType[T] {
	return &MsgType[T]{
		m: New(),
	}
}

func NewTypeTo[T any](to ...time.Duration) *MsgType[T] {
	return &MsgType[T]{
		m: NewTo(to...),
	}
}

func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	defer func() {
		if e := recover(); e != nil {
			panic(fmt.Sprintf("Push_tag(%s,%v) > %v", Tag, Data, e))
		}
	}()
	m.m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *MsgType[T]) PushLock_tag(Tag string, Data T) {
	defer func() {
		if e := recover(); e != nil {
			panic(fmt.Sprintf("PushLock_tag(%s,%v) > %v", Tag, Data, e))
		}
	}()
	m.m.PushLock(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *MsgType[T]) ClearAll() {
	m.m.ClearAll()
}

func (m *MsgType[T]) Pull_tag_chan(key string, size int, ctx context.Context) <-chan T {
	var ch = make(chan T, size)
	m.m.Register(func(data any) bool {
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			select {
			case <-ctx.Done():
				close(ch)
				return true
			default:
				if len(ch) == size {
					<-ch
				}
				ch <- d.Data.(T)
				return false
			}
		}
		return false
	})
	return ch
}

func (m *MsgType[T]) Pull_tag_only(key string, f func(T) (disable bool)) {
	m.m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			return f(d.Data.(T))
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) {
	m.m.Register(func(data any) (disable bool) {
		if d, ok := data.(Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data.(T))
			}
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async_only(key string, f func(T) (disable bool)) {
	var disable = signal.Init()

	m.m.Register_front(func(data any) bool {
		if !disable.Islive() {
			return true
		}
		if d, ok := data.(Msgq_tag_data); ok && d.Tag == key {
			go func() {
				if f(d.Data.(T)) {
					disable.Done()
				}
			}()
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async(func_map map[string]func(T) (disable bool)) {
	var disable = signal.Init()

	m.m.Register_front(func(data any) bool {
		if !disable.Islive() {
			return true
		}
		if d, ok := data.(Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					if f(d.Data.(T)) {
						disable.Done()
					}
				}()
			}
		}
		return false
	})
}
