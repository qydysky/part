package part

import (
	"container/list"
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	signal "github.com/qydysky/part/signal"
	sync "github.com/qydysky/part/sync"
)

type Msgq struct {
	to             []time.Duration
	funcs          *list.List
	someNeedRemove atomic.Int32
	lock           sync.RWMutex
	runTag         sync.Map
}

type FuncMap map[string]func(any) (disable bool)

func New() *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	return m
}

func NewTo(to time.Duration) *Msgq {
	fmt.Println("Warn: NewTo is slow, consider New")
	m := new(Msgq)
	m.funcs = list.New()
	if to != 0 {
		m.to = append(m.to, to)
	} else {
		m.to = append(m.to, time.Second*30)
	}
	return m
}

func (m *Msgq) Register(f func(any) (disable bool)) {
	ul := m.lock.Lock(m.to...)()
	m.funcs.PushBack(f)
	ul()
}

func (m *Msgq) Register_front(f func(any) (disable bool)) {
	ul := m.lock.Lock(m.to...)()
	m.funcs.PushFront(f)
	ul()
}

func (m *Msgq) Push(msg any) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	var removes []*list.Element

	ul := m.lock.RLock(m.to...)()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(any) bool)(msg); disable {
			m.someNeedRemove.Add(1)
			removes = append(removes, el)
		}
	}
	ul()

	if len(removes) != 0 {
		ul := m.lock.Lock(m.to...)()
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

	ul := m.lock.Lock(m.to...)()
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

	ul := m.lock.Lock(m.to...)()
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
	to             []time.Duration
	funcs          *list.List
	someNeedRemove atomic.Int32
	lock           sync.RWMutex
	runTag         sync.Map
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

func NewTypeTo[T any](to time.Duration) *MsgType[T] {
	fmt.Println("Warn: NewTypeTo[T any] is slow, consider NewType[T any]")
	m := new(MsgType[T])
	m.funcs = list.New()
	if to != 0 {
		m.to = append(m.to, to)
	} else {
		m.to = append(m.to, time.Second*30)
	}
	return m
}

func (m *MsgType[T]) push(msg MsgType_tag_data[T]) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	var removes []*list.Element

	ul := m.lock.RLock(m.to...)()
	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(MsgType_tag_data[T]) bool)(msg); disable {
			m.someNeedRemove.Add(1)
			removes = append(removes, el)
		}
	}
	ul()

	if len(removes) != 0 {
		ul := m.lock.Lock(m.to...)()
		m.someNeedRemove.Add(-int32(len(removes)))
		for i := 0; i < len(removes); i++ {
			m.funcs.Remove(removes[i])
		}
		ul()
	}
}

func (m *MsgType[T]) pushLock(msg MsgType_tag_data[T]) {
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	ul := m.lock.Lock(m.to...)()
	defer ul()

	var removes []*list.Element

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if disable := el.Value.(func(MsgType_tag_data[T]) bool)(msg); disable {
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

func (m *MsgType[T]) register(f func(MsgType_tag_data[T]) (disable bool)) {
	ul := m.lock.Lock(m.to...)()
	m.funcs.PushBack(f)
	ul()
}

func (m *MsgType[T]) register_front(f func(MsgType_tag_data[T]) (disable bool)) {
	ul := m.lock.Lock(m.to...)()
	m.funcs.PushFront(f)
	ul()
}

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
	for m.someNeedRemove.Load() != 0 {
		time.Sleep(time.Millisecond)
		runtime.Gosched()
	}

	ul := m.lock.Lock(m.to...)()
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
