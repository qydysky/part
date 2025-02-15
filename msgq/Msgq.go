package part

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	lmt "fmt"
	"go/build"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	psync "github.com/qydysky/part/sync"
)

var ErrRunTO = errors.New(`ErrRunTO`)

type Msgq struct {
	to    []time.Duration
	funcs *list.List

	someNeedRemove atomic.Bool
	allNeedRemove  atomic.Bool
	lock           psync.RWMutex
	PanicFunc      func(any)
}

type msgqItem struct {
	disable atomic.Bool
	f       *func(any) (disable bool)
}

type FuncMap map[string]func(any) (disable bool)
type FuncMapType[T any] map[string]func(T) (disable bool)

// to[0]:timeout to wait to[1]:timeout to run
func New(to ...time.Duration) *Msgq {
	m := new(Msgq)
	m.funcs = list.New()
	m.to = to
	if len(m.to) > 0 {
		m.lock.RecLock(10, to[0], to[0])
	}
	return m
}

func (m *Msgq) TOPanicFunc(f func(any)) {
	m.PanicFunc = f
	m.lock.PanicFunc = f
}

func (m *Msgq) register(mp *msgqItem, f func(v any) *list.Element) (cancel func()) {
	defer m.lock.Lock()()

	if m.allNeedRemove.Load() {
		m.removeDisable(m.someNeedRemove.CompareAndSwap(false, true), true)
	}

	f(mp)
	return func() {
		mp.disable.Store(true)
		m.removeDisable(m.someNeedRemove.CompareAndSwap(false, true), false)
	}
}

func (m *Msgq) Register(f func(any) (disable bool)) (cancel func()) {
	return m.register(&msgqItem{
		f: &f,
	}, m.funcs.PushBack)
}

func (m *Msgq) RegisterFront(f func(any) (disable bool)) (cancel func()) {
	return m.register(&msgqItem{
		f: &f,
	}, m.funcs.PushFront)
}

func (m *Msgq) push(msg any, isLock bool) {
	if isLock {
		defer m.lock.Lock()()
	} else {
		defer m.lock.RLock()()
	}

	if m.allNeedRemove.Load() {
		m.removeDisable(true, isLock)
		if !isLock {
			return
		}
	}

	for el := m.funcs.Front(); el != nil; el = el.Next() {
		if mi := el.Value.(*msgqItem); !mi.disable.Load() && (*mi.f)(msg) {
			mi.disable.Store(true)
			m.removeDisable(m.someNeedRemove.CompareAndSwap(false, true), isLock)
		}
	}
}

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *Msgq) Push(msg any) {
	m.push(msg, false)
}

// 不能在由Push*调用的Pull中以同步方式使用
func (m *Msgq) PushLock(msg any) {
	m.push(msg, true)
}

func (m *Msgq) ClearAll() {
	m.allNeedRemove.Store(true)
}

func (m *Msgq) removeDisable(sig bool, isLock bool) {
	if sig {
		f := func() {
			if !isLock {
				defer m.lock.Lock()()
			}
			all := m.allNeedRemove.Swap(false)
			for el := m.funcs.Front(); el != nil; el = el.Next() {
				mi := el.Value.(*msgqItem)
				if all || mi.disable.Load() {
					m.funcs.Remove(el)
				}
			}
			m.someNeedRemove.Store(false)
		}
		if isLock {
			f()
		} else {
			go f()
		}
	}
}

func (m *Msgq) panicFunc(s any) {
	if m.PanicFunc != nil {
		m.PanicFunc(s)
	} else {
		panic(s)
	}
}

func (m *Msgq) PushingTO(info string, callTree *string) (fin func()) {
	if len(m.to) > 1 {
		to := time.AfterFunc(m.to[1], func() {
			m.panicFunc(errors.Join(ErrRunTO, lmt.Errorf("%v:%v", info, *callTree)))
		})
		return func() {
			to.Stop()
		}
	}
	return func() {}
}

type Msgq_tag_data struct {
	Tag  string
	Data any
}

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *Msgq) Push_tag(Tag string, Data any) {
	defer m.PushingTO(lmt.Sprintf("\nPush_tag(`%v`)", Tag), getCall(1))()
	m.Push(&Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

// 不能在由Push*调用的Pull中以同步方式使用
func (m *Msgq) PushLock_tag(Tag string, Data any) {
	defer m.PushingTO(lmt.Sprintf("\nPushLock_tag(`%v`)", Tag), getCall(1))()
	m.PushLock(&Msgq_tag_data{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) Pull_tag_chan(key string, size int, ctx context.Context) (cancel func(), ch <-chan any) {
	var c = make(chan any, size)
	return m.Register(func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); ok && d.Tag == key {
			select {
			case <-ctx.Done():
				close(c)
				return true
			default:
				empty := false
				for !empty {
					select {
					case <-c:
					default:
						c <- d.Data
						empty = true
					}
				}
			}
		}
		return false
	}), c
}

func (m *Msgq) Pull_tag_only(key string, f func(any) (disable bool)) (cancel func()) {
	return m.Register(func(data any) (disable bool) {
		if d, ok := data.(*Msgq_tag_data); ok && d.Tag == key {
			return f(d.Data)
		}
		return false
	})
}

func (m *Msgq) Pull_tag(func_map map[string]func(any) (disable bool)) (cancel func()) {
	return m.Register(func(data any) (disable bool) {
		if d, ok := data.(*Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				return f(d.Data)
			}
		}
		return false
	})
}

func (m *Msgq) Pull_tag_async_only(key string, f func(any) (disable bool)) (cancel func()) {
	var disable bool
	return m.RegisterFront(func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); !disable && ok && d.Tag == key {
			go func() {
				disable = f(d.Data)
			}()
		}
		return disable
	})
}

func (m *Msgq) Pull_tag_async(func_map map[string]func(any) (disable bool)) (cancel func()) {
	var disable bool
	return m.RegisterFront(func(data any) bool {
		if d, ok := data.(*Msgq_tag_data); ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					disable = f(d.Data)
				}()
			}
		}
		return disable
	})
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

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	m.m.Push(&MsgType_tag_data[T]{
		Tag:  Tag,
		Data: &Data,
	})
}

// 不能在由Push*调用的Pull中以同步方式使用
func (m *MsgType[T]) PushLock_tag(Tag string, Data T) {
	m.m.PushLock(&MsgType_tag_data[T]{
		Tag:  Tag,
		Data: &Data,
	})
}

func (m *MsgType[T]) Pull_tag_chan(key string, size int, ctx context.Context) (cancel func(), ch <-chan T) {
	var c = make(chan T, size)
	return m.m.Register(func(data any) bool {
		if data1, ok := data.(*MsgType_tag_data[T]); ok {
			if data1.Tag == key {
				select {
				case <-ctx.Done():
					close(c)
					return true
				default:
					empty := false
					for !empty {
						select {
						case <-c:
						default:
							c <- *data1.Data
							empty = true
						}
					}
				}
			}
		}
		return false
	}), c
}

func (m *MsgType[T]) Pull_tag_only(key string, f func(T) (disable bool)) (cancel func()) {
	return m.m.Register(func(data any) (disable bool) {
		if data1, ok := data.(*MsgType_tag_data[T]); ok && data1.Tag == key {
			return f(*data1.Data)
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) (cancel func()) {
	return m.m.Register(func(data any) (disable bool) {
		if data1, ok := data.(*MsgType_tag_data[T]); ok {
			if f, ok := func_map[data1.Tag]; ok {
				return f(*data1.Data)
			}
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async_only(key string, f func(T) (disable bool)) (cancel func()) {
	var disabled atomic.Bool
	return m.m.RegisterFront(func(data any) bool {
		if d, ok := data.(*MsgType_tag_data[T]); !disabled.Load() && ok && d.Tag == key {
			go func() {
				if !disabled.Load() {
					disabled.Store(f(*d.Data))
				}
			}()
		}
		return disabled.Load()
	})
}

func (m *MsgType[T]) Pull_tag_async(func_map map[string]func(T) (disable bool)) (cancel func()) {
	var disabled atomic.Bool
	return m.m.RegisterFront(func(data any) bool {
		if d, ok := data.(*MsgType_tag_data[T]); !disabled.Load() && ok {
			if f, ok := func_map[d.Tag]; ok {
				go func() {
					if !disabled.Load() {
						disabled.Store(f(*d.Data))
					}
				}()
			}
		}
		return disabled.Load()
	})
}

func getCall(i int) (calls *string) {
	var cs string
	for i += 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, build.Default.GOROOT) {
			break
		} else {
			cs += fmt.Sprintf("\ncall by %s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	if cs == "" {
		cs += fmt.Sprintln("\ncall by goroutine")
	}
	return &cs
}
