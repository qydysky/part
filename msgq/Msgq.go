package part

import (
	"container/list"
	"context"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	signal "github.com/qydysky/part/signal"
	psync "github.com/qydysky/part/sync"
)

type Msgq struct {
	to    []time.Duration
	funcs *list.List

	call       atomic.Int64
	removeList []*list.Element
	removelock psync.RWMutex

	lock   psync.RWMutex
	runTag psync.Map
}

type FuncMap map[string]func(any) (disable bool)

func New() *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	return m
}

func NewTo(waitTo time.Duration, runTo ...time.Duration) *Msgq {
	fmt.Println("Warn: NewTo is slow, consider New")
	m := new(Msgq)
	m.funcs = list.New()
	m.to = append([]time.Duration{waitTo}, runTo...)
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
	isfirst := m.call.Add(1)

	ul := m.lock.RLock(m.to...)
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			rul := m.removelock.Lock()
			m.removeList = append(m.removeList, el)
			rul()
		}
	}
	ul()

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
}

func (m *Msgq) PushLock(msg any) {
	isfirst := m.call.Add(1)

	ul := m.lock.Lock(m.to...)
	defer ul()

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			rul := m.removelock.Lock()
			m.removeList = append(m.removeList, el)
			rul()
		}
	}

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
}

func (m *Msgq) ClearAll() {
	isfirst := m.call.Add(1)

	rul := m.removelock.Lock()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		m.removeList = append(m.removeList, el)
	}
	rul()

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
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
	m.Push(Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
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
				for len(ch) != 0 {
					<-ch
				}
				ch <- d.Data
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
	to    []time.Duration
	funcs *list.List

	call       atomic.Int64
	removeList []*list.Element
	removelock psync.RWMutex

	lock   psync.RWMutex
	runTag psync.Map
}

type MsgType_tag_data[T any] struct {
	Tag  string
	Data T
}

func NewType[T any]() *MsgType[T] {
	m := new(MsgType[T])
	m.funcs = list.New()
	return m
}

func NewTypeTo[T any](waitTo time.Duration, runTo ...time.Duration) *MsgType[T] {
	fmt.Println("Warn: NewTypeTo[T any] is slow, consider NewType[T any]")
	m := new(MsgType[T])
	m.funcs = list.New()
	m.to = append([]time.Duration{waitTo}, runTo...)
	return m
}

func (m *MsgType[T]) push(msg MsgType_tag_data[T]) {
	isfirst := m.call.Add(1)

	ul := m.lock.RLock(m.to...)
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(MsgType_tag_data[T]) bool)(msg); disable {
			rul := m.removelock.Lock()
			m.removeList = append(m.removeList, el)
			rul()
		}
	}
	ul()

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
}

func (m *MsgType[T]) pushLock(msg MsgType_tag_data[T]) {
	isfirst := m.call.Add(1)

	ul := m.lock.Lock(m.to...)
	defer ul()

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(MsgType_tag_data[T]) bool)(msg); disable {
			rul := m.removelock.Lock()
			m.removeList = append(m.removeList, el)
			rul()
		}
	}

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
}

func (m *MsgType[T]) register(f func(MsgType_tag_data[T]) (disable bool)) {
	ul := m.lock.Lock(m.to...)
	m.funcs.PushBack(f)
	ul()
}

func (m *MsgType[T]) register_front(f func(MsgType_tag_data[T]) (disable bool)) {
	ul := m.lock.Lock(m.to...)
	m.funcs.PushFront(f)
	ul()
}

// 不能放置在由PushLock_tag调用的同步Pull中
func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	if len(m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.runTag.Store(ptr, "[T]Push_tag(`"+Tag+"`,...)")
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
	m.push(MsgType_tag_data[T]{
		Tag:  Tag,
		Data: Data,
	})
}

// 不能放置在由Push_tag、PushLock_tag调用的同步Pull中
func (m *MsgType[T]) PushLock_tag(Tag string, Data T) {
	if len(m.to) > 0 {
		ptr := uintptr(unsafe.Pointer(&Data))
		m.runTag.Store(ptr, "[T]PushLock_tag(`"+Tag+"`,...)")
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
	m.pushLock(MsgType_tag_data[T]{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *MsgType[T]) ClearAll() {
	isfirst := m.call.Add(1)

	rul := m.removelock.Lock()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		m.removeList = append(m.removeList, el)
	}
	rul()

	if isfirst == 1 {
		rul := m.removelock.Lock()
		for i := 0; i < len(m.removeList); i++ {
			m.funcs.Remove(m.removeList[i])
		}
		rul()
	}

	m.call.Add(-1)
}

func (m *MsgType[T]) Pull_tag_chan(key string, size int, ctx context.Context) <-chan T {
	var ch = make(chan T, size)
	m.register(func(data MsgType_tag_data[T]) bool {
		if data.Tag == key {
			select {
			case <-ctx.Done():
				close(ch)
				return true
			default:
				for len(ch) != 0 {
					<-ch
				}
				ch <- data.Data
			}
		}
		return false
	})
	return ch
}

func (m *MsgType[T]) Pull_tag_only(key string, f func(T) (disable bool)) {
	m.register(func(data MsgType_tag_data[T]) (disable bool) {
		if data.Tag == key {
			return f(data.Data)
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) {
	m.register(func(data MsgType_tag_data[T]) (disable bool) {
		if f, ok := func_map[data.Tag]; ok {
			return f(data.Data)
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async_only(key string, f func(T) (disable bool)) {
	var disable = signal.Init()

	m.register_front(func(data MsgType_tag_data[T]) bool {
		if !disable.Islive() {
			return true
		}
		if data.Tag == key {
			go func() {
				if f(data.Data) {
					disable.Done()
				}
			}()
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async(func_map map[string]func(T) (disable bool)) {
	var disable = signal.Init()

	m.register_front(func(data MsgType_tag_data[T]) bool {
		if !disable.Islive() {
			return true
		}
		if f, ok := func_map[data.Tag]; ok {
			go func() {
				if f(data.Data) {
					disable.Done()
				}
			}()
		}
		return false
	})
}
