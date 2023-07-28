package part

import (
	"container/list"
	"context"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	psync "github.com/qydysky/part/sync"
)

type Msgq struct {
	to    []time.Duration
	funcs *list.List

	someNeedRemove atomic.Bool
	lock           psync.RWMutex
	runTag         psync.Map
}

type msgqItem struct {
	running atomic.Int32
	disable atomic.Bool
	f       *func(any) (disable bool)
}

type FuncMap map[string]func(any) (disable bool)

// to[0]:timeout to wait to[1]:timeout to run
func New(to ...time.Duration) *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	m.to = to
	return m
}

func (m *Msgq) register(mp *msgqItem) {
	ul := m.lock.Lock()
	m.funcs.PushBack(mp)
	ul()
}

func (m *Msgq) Register(f func(any) (disable bool)) {
	m.register(&msgqItem{
		f: &f,
	})
}

func (m *Msgq) register_front(mp *msgqItem) {
	ul := m.lock.Lock()
	m.funcs.PushFront(mp)
	ul()
}

func (m *Msgq) Register_front(f func(any) (disable bool)) {
	m.register_front(&msgqItem{
		f: &f,
	})
}

func (m *Msgq) Push(msg any) {
	defer m.removeDisable()
	ul := m.lock.RLock(m.to...)
	defer ul()

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		mi := el.Value.(*msgqItem)
		if mi.disable.Load() {
			continue
		}
		mi.running.Add(1)
		if disable := (*mi.f)(msg); disable {
			mi.disable.Store(true)
			m.someNeedRemove.Store(true)
		}
		mi.running.Add(-1)
	}
}

func (m *Msgq) PushLock(msg any) {
	defer m.removeDisable()
	ul := m.lock.Lock(m.to...)
	defer ul()

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		mi := el.Value.(*msgqItem)
		if mi.disable.Load() {
			continue
		}
		mi.running.Add(1)
		if disable := (*mi.f)(msg); disable {
			mi.disable.Store(true)
			m.someNeedRemove.Store(true)
		}
		mi.running.Add(-1)
	}
}

func (m *Msgq) ClearAll() {
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		mi := el.Value.(*msgqItem)
		mi.disable.Store(true)
		m.someNeedRemove.Store(true)
	}
}

func (m *Msgq) removeDisable() {
	if !m.someNeedRemove.CompareAndSwap(true, false) {
		return
	}

	ul := m.lock.Lock(m.to...)
	defer ul()

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		mi := el.Value.(*msgqItem)
		if mi.disable.Load() && mi.running.Load() == 0 {
			m.funcs.Remove(el)
		}
	}
}

type Msgq_tag_data struct {
	Tag  string
	Data any
}

// 不能放置在由PushLock_tag调用的同步Pull中
func (m *Msgq) Push_tag(Tag string, Data any) {
	if len(m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.runTag.Store(ptr, "Push_tag(`"+Tag+"`,...)")
		defer func() {
			if e := recover(); e != nil {
				m.runTag.Range(func(key, value any) bool {
					if key == ptr {
						fmt.Printf("%v panic > %v\n", value, e)
					} else {
						fmt.Printf("%v running\n", value)
					}
					return true
				})
				m.runTag.ClearAll()
				panic(e)
			}
			m.runTag.Delete(ptr)
		}()
	}
	{
		/*
			m.m.Push(Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			})
		*/
		defer m.removeDisable()
		ul := m.lock.RLock(m.to...)
		defer ul()

		for el := m.funcs.Front(); el != nil; el = el.Next() {
			mi := el.Value.(*msgqItem)
			if mi.disable.Load() {
				continue
			}
			mi.running.Add(1)
			if disable := (*mi.f)(&Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			}); disable {
				mi.disable.Store(true)
				m.someNeedRemove.Store(true)
			}
			mi.running.Add(-1)
		}
	}
}

// 不能放置在由Push_tag、PushLock_tag调用的同步Pull中
func (m *Msgq) PushLock_tag(Tag string, Data any) {
	if len(m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.runTag.Store(ptr, "PushLock_tag(`"+Tag+"`,...)")
		defer func() {
			if e := recover(); e != nil {
				m.runTag.Range(func(key, value any) bool {
					if key == ptr {
						fmt.Printf("%v panic > %v\n", value, e)
					} else {
						fmt.Printf("%v running\n", value)
					}
					return true
				})
				m.runTag.ClearAll()
				panic(e)
			}
			m.runTag.Delete(ptr)
		}()
	}
	{
		/*
			m.m.PushLock(Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			})
		*/
		defer m.removeDisable()
		ul := m.lock.Lock(m.to...)
		defer ul()

		for el := m.funcs.Front(); el != nil; el = el.Next() {
			mi := el.Value.(*msgqItem)
			if mi.disable.Load() {
				continue
			}
			mi.running.Add(1)
			if disable := (*mi.f)(&Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			}); disable {
				mi.disable.Store(true)
				m.someNeedRemove.Store(true)
			}
			mi.running.Add(-1)
		}
	}
}

func (m *Msgq) Pull_tag_chan(key string, size int, ctx context.Context) <-chan any {
	var ch = make(chan any, size)
	var f1 = func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); ok && d.Tag == key {
			select {
			case <-ctx.Done():
				close(ch)
				return true
			default:
				for len(ch) != 0 {
					<-ch
				}
				ch <- d.Data
			}
		}
		return false
	}
	m.register_front(&msgqItem{
		f: &f1,
	})
	return ch
}

func (m *Msgq) Pull_tag_only(key string, f func(any) (disable bool)) {
	var f1 = func(data any) (disable bool) {
		if d, ok := data.(*Msgq_tag_data); ok && d.Tag == key {
			return f(d.Data)
		}
		return false
	}
	m.register_front(&msgqItem{
		f: &f1,
	})
}

func (m *Msgq) Pull_tag(func_map map[string]func(any) (disable bool)) {
	var f1 = func(data any) (disable bool) {
		if d, ok := data.(*Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data)
			}
		}
		return false
	}
	m.register_front(&msgqItem{
		f: &f1,
	})
}

func (m *Msgq) Pull_tag_async_only(key string, f func(any) (disable bool)) {
	var mi = msgqItem{}
	var f1 = func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); ok {
			go func() {
				if f(d.Data) {
					mi.disable.Store(true)
					m.someNeedRemove.Store(true)
				}
			}()
		}
		return false
	}
	mi.f = &f1
	m.register_front(&mi)
}

func (m *Msgq) Pull_tag_async(func_map map[string]func(any) (disable bool)) {
	var mi = msgqItem{}
	var f = func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					if f(d.Data) {
						mi.disable.Store(true)
						m.someNeedRemove.Store(true)
					}
				}()
			}
		}
		return false
	}
	mi.f = &f
	m.register_front(&mi)
}

type MsgType[T any] struct {
	m *Msgq
}

type MsgType_tag_data[T any] struct {
	Tag  string
	Data *T
}

// to[0]:timeout to wait to[1]:timeout to run
func NewType[T any](to ...time.Duration) *MsgType[T] {
	return &MsgType[T]{m: New(to...)}
}

func (m *MsgType[T]) ClearAll() {
	m.m.ClearAll()
}

// 不能放置在由PushLock_tag调用的同步Pull中
func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	if len(m.m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.m.runTag.Store(ptr, "Push_tag(`"+Tag+"`,...)")
		defer func() {
			if e := recover(); e != nil {
				m.m.runTag.Range(func(key, value any) bool {
					if key == ptr {
						fmt.Printf("%v panic > %v\n", value, e)
					} else {
						fmt.Printf("%v running\n", value)
					}
					return true
				})
				m.m.runTag.ClearAll()
				panic(e)
			}
			m.m.runTag.Delete(ptr)
		}()
	}
	{
		/*
			m.m.Push(Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			})
		*/
		defer m.m.removeDisable()
		ul := m.m.lock.RLock(m.m.to...)
		defer ul()

		for el := m.m.funcs.Front(); el != nil; el = el.Next() {
			mi := el.Value.(*msgqItem)
			if mi.disable.Load() {
				continue
			}
			mi.running.Add(1)
			if disable := (*mi.f)(&MsgType_tag_data[T]{
				Tag:  Tag,
				Data: &Data,
			}); disable {
				mi.disable.Store(true)
				m.m.someNeedRemove.Store(true)
			}
			mi.running.Add(-1)
		}
	}
}

// 不能放置在由Push_tag、PushLock_tag调用的同步Pull中
func (m *MsgType[T]) PushLock_tag(Tag string, Data T) {
	if len(m.m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.m.runTag.Store(ptr, "PushLock_tag(`"+Tag+"`,...)")
		defer func() {
			if e := recover(); e != nil {
				m.m.runTag.Range(func(key, value any) bool {
					if key == ptr {
						fmt.Printf("%v panic > %v\n", value, e)
					} else {
						fmt.Printf("%v running\n", value)
					}
					return true
				})
				m.m.runTag.ClearAll()
				panic(e)
			}
			m.m.runTag.Delete(ptr)
		}()
	}
	{
		/*
			m.m.PushLock(Msgq_tag_data{
				Tag:  Tag,
				Data: Data,
			})
		*/
		defer m.m.removeDisable()
		ul := m.m.lock.Lock(m.m.to...)
		defer ul()

		for el := m.m.funcs.Front(); el != nil; el = el.Next() {
			mi := el.Value.(*msgqItem)
			if mi.disable.Load() {
				continue
			}
			mi.running.Add(1)
			if disable := (*mi.f)(&MsgType_tag_data[T]{
				Tag:  Tag,
				Data: &Data,
			}); disable {
				mi.disable.Store(true)
				m.m.someNeedRemove.Store(true)
			}
			mi.running.Add(-1)
		}
	}
}

func (m *MsgType[T]) Pull_tag_chan(key string, size int, ctx context.Context) <-chan T {
	var ch = make(chan T, size)
	var f = func(data any) bool {
		if data1, ok := data.(*MsgType_tag_data[T]); ok {
			if data1.Tag == key {
				select {
				case <-ctx.Done():
					close(ch)
					return true
				default:
					for len(ch) != 0 {
						<-ch
					}
					ch <- *data1.Data
				}
			}
		}
		return false
	}
	m.m.register(&msgqItem{
		f: &f,
	})
	return ch
}

func (m *MsgType[T]) Pull_tag_only(key string, f func(T) (disable bool)) {
	var f1 = func(data any) (disable bool) {
		if data1, ok := data.(*MsgType_tag_data[T]); ok {
			if data1.Tag == key {
				return f(*data1.Data)
			}
		}
		return false
	}
	m.m.register(&msgqItem{
		f: &f1,
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) {
	var f = func(data any) (disable bool) {
		if data1, ok := data.(*MsgType_tag_data[T]); ok {
			if f, ok := func_map[data1.Tag]; ok {
				return f(*data1.Data)
			}
		}
		return false
	}
	m.m.register(&msgqItem{
		f: &f,
	})
}

func (m *MsgType[T]) Pull_tag_async_only(key string, f func(T) (disable bool)) {
	var mi = msgqItem{}
	var f1 = func(data any) bool {
		if d, ok := data.(*MsgType_tag_data[T]); ok {
			go func() {
				if f(*d.Data) {
					mi.disable.Store(true)
					m.m.someNeedRemove.Store(true)
				}
			}()
		}
		return false
	}
	mi.f = &f1
	m.m.register_front(&mi)
}

func (m *MsgType[T]) Pull_tag_async(func_map map[string]func(T) (disable bool)) {
	var mi = msgqItem{}
	var f = func(data any) bool {
		if d, ok := data.(*MsgType_tag_data[T]); ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					if f(*d.Data) {
						mi.disable.Store(true)
						m.m.someNeedRemove.Store(true)
					}
				}()
			}
		}
		return false
	}
	mi.f = &f
	m.m.register_front(&mi)
}
