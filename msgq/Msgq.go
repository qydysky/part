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

type msgqItem[T any] struct {
	callTree *string
	disable  atomic.Bool
	f        func(T) (disable bool)
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

func (m *Msgq) register[T any](mp *msgqItem[T], f func(v any) *list.Element) (cancel func()) {
	defer m.lock.Lock()()

	if m.allNeedRemove.Load() {
		m.removeDisable[T](m.someNeedRemove.CompareAndSwap(false, true), true)
	}

	f(mp)
	return func() {
		mp.disable.Store(true)
		m.removeDisable[T](m.someNeedRemove.CompareAndSwap(false, true), false)
	}
}

func (m *Msgq) Register[T any](f func(T) (disable bool)) (cancel func()) {
	return m.register(&msgqItem[T]{
		callTree: getCall(),
		f:        f,
	}, m.funcs.PushBack)
}

func (m *Msgq) RegisterFront[T any](f func(T) (disable bool)) (cancel func()) {
	return m.register(&msgqItem[T]{
		callTree: getCall(),
		f:        f,
	}, m.funcs.PushFront)
}

// if push[T]'s generic is not the same as register[T]'s generic, register f with be skip
func (m *Msgq) push[T any](msg T, isLock bool) {
	if isLock {
		defer m.lock.Lock()()
	} else {
		defer m.lock.RLock()()
	}

	if m.allNeedRemove.Load() {
		m.removeDisable[T](true, isLock)
		if !isLock {
			return
		}
	}

	if len(m.to) > 1 {
		var pullCalltree atomic.Pointer[string]
		if isLock {
			defer m.pushingTO("\nlock pushing", getCall(), pullCalltree.Load)()
		} else {
			defer m.pushingTO("\nrlock pushing", getCall(), pullCalltree.Load)()
		}

		for el := m.funcs.Front(); el != nil; el = el.Next() {
			if mi, ok := el.Value.(*msgqItem[T]); ok && !mi.disable.Load() {
				pullCalltree.Store(mi.callTree)
				if mi.f(msg) {
					mi.disable.Store(true)
					m.removeDisable[T](m.someNeedRemove.CompareAndSwap(false, true), isLock)
				}
			}
		}
	} else {
		for el := m.funcs.Front(); el != nil; el = el.Next() {
			if mi, ok := el.Value.(*msgqItem[T]); ok && !mi.disable.Load() {
				if mi.f(msg) {
					mi.disable.Store(true)
					m.removeDisable[T](m.someNeedRemove.CompareAndSwap(false, true), isLock)
				}
			}
		}
	}
}

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *Msgq) Push[T any](msg T) {
	m.push(msg, false)
}

// 不能在由Push*调用的Pull中以同步方式使用
func (m *Msgq) PushLock[T any](msg T) {
	m.push(msg, true)
}

func (m *Msgq) ClearAll() {
	m.allNeedRemove.Store(true)
}

func (m *Msgq) removeDisable[T any](sig bool, isLock bool) {
	if sig {
		f := func() {
			if !isLock {
				defer m.lock.Lock()()
			}
			all := m.allNeedRemove.Swap(false)
			for el := m.funcs.Front(); el != nil; el = el.Next() {
				if mi, ok := el.Value.(*msgqItem[T]); all || (ok && mi.disable.Load()) {
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

func (m *Msgq) pushingTO(info string, callTree *string, pullCalltree func() *string) (fin func()) {
	// if len(m.to) > 1 {
	to := time.AfterFunc(m.to[1], func() {
		if pullTree := pullCalltree(); pullTree != nil {
			m.panicFunc(errors.Join(ErrRunTO, lmt.Errorf("%v:%v\nrunning pull:%v", info, *callTree, *pullTree)))
		} else {
			m.panicFunc(errors.Join(ErrRunTO, lmt.Errorf("%v:%v\nrunning pull:none", info, *callTree)))
		}
	})
	return func() {
		to.Stop()
	}
	// }
	// return func() {}
}

type Msgq_tag_data[T any] struct {
	Tag  string
	Data T
}

func (t Msgq_tag_data[T]) String() string {
	return fmt.Sprintf("tag:%v", t.Tag)
}

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *Msgq) Push_tag[T any](Tag string, Data T) {
	// if len(m.to) > 1 {
	// 	defer m.PushingTO(lmt.Sprintf("\nPush_tag(`%v`)", Tag), getCall(1))()
	// }
	m.Push(&Msgq_tag_data[T]{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) PushSign(Tag string) {
	m.Push_tag(Tag, struct{}{})
}

// 不能在由Push*调用的Pull中以同步方式使用
//
// async类的Pull将会创建协程处理并退出，可能不会按预期工作
func (m *Msgq) PushLock_tag[T any](Tag string, Data T) {
	// if len(m.to) > 1 {
	// 	defer m.PushingTO(lmt.Sprintf("\nPushLock_tag(`%v`)", Tag), getCall(1))()
	// }
	m.PushLock(&Msgq_tag_data[T]{
		Tag:  Tag,
		Data: Data,
	})
}

func (m *Msgq) PushLockSign(Tag string) {
	m.PushLock_tag(Tag, struct{}{})
}

func (m *Msgq) Pull_tag_chan[T any](key string, size int, ctx context.Context) (cancel func(), ch <-chan T) {
	c := make(chan T, size)
	ctx, cancelC := context.WithCancel(ctx)
	unreg := m.Register(func(data *Msgq_tag_data[T]) bool {
		if data.Tag == key {
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
						c <- data.Data
						empty = true
					}
				}
			}
		}
		return false
	})
	return func() {
		select {
		case <-ctx.Done():
		default:
			cancelC()
			unreg()
			close(c)
		}
	}, c
}

func (m *Msgq) PullSignChan(Tag string, ctx context.Context) (cancel func(), ch <-chan struct{}) {
	return m.Pull_tag_chan[struct{}](Tag, 2, ctx)
}

func (m *Msgq) Pull_tag_only[T any](key string, f func(T) (disable bool)) (cancel func()) {
	return m.Register(func(data *Msgq_tag_data[T]) (disable bool) {
		if data.Tag == key {
			return f(data.Data)
		}
		return false
	})
}

func (m *Msgq) PullSignOnly(Tag string, f func(_ struct{}) (disable bool)) (cancel func()) {
	return m.Pull_tag_only(Tag, f)
}

func (m *Msgq) Pull_tag[T any](func_map map[string]func(T) (disable bool)) (cancel func()) {
	return m.Register(func(data *Msgq_tag_data[T]) (disable bool) {
		if f, ok := func_map[data.Tag]; ok {
			return f(data.Data)
		}
		return false
	})
}

func (m *Msgq) PullSign(func_map map[string]func(_ struct{}) (disable bool)) (cancel func()) {
	return m.Register(func(data *Msgq_tag_data[struct{}]) (disable bool) {
		if f, ok := func_map[data.Tag]; ok {
			return f(data.Data)
		}
		return false
	})
}

type Register struct {
	mq     *Msgq
	cancel atomic.Pointer[func()]
}

func (m *Register) Tag[T any](key string, f func(T) (disable bool)) {
	cancel := m.mq.Register(func(data *Msgq_tag_data[T]) (disable bool) {
		if data.Tag == key {
			disable = f(data.Data)
		}
		if disable {
			(*m.cancel.Load())()
		}
		return
	})
	{
		var pre *func()
		var cur = func() {
			if pre != nil {
				(*pre)()
			}
			cancel()
		}
		pre = m.cancel.Swap(&cur)
	}
}

func (m *Register) TagSign(key string, f func(_ struct{}) (disable bool)) {
	m.Tag(key, f)
}

func (m *Register) TagAsync[T any](key string, f func(T) (disable bool)) {
	var disabled atomic.Bool
	cancel := m.mq.RegisterFront(func(data *Msgq_tag_data[T]) bool {
		if !disabled.Load() && data.Tag == key {
			go func() {
				disabled.CompareAndSwap(false, f(data.Data))
			}()
		}
		return disabled.Load()
	})
	{
		var pre *func()
		var cur = func() {
			if pre != nil {
				(*pre)()
			}
			cancel()
		}
		pre = m.cancel.Swap(&cur)
	}
}

func (m *Register) TagSignAsync(key string, f func(_ struct{}) (disable bool)) {
	m.TagAsync(key, f)
}

func (m *Msgq) Pull_tags(batchTag func(fc *Register)) (cancel func()) {
	reg := &Register{mq: m}
	batchTag(reg)
	return *reg.cancel.Load()
}

func (m *Msgq) Pull_tag_async_only[T any](key string, f func(T) (disable bool)) (cancel func()) {
	var disable atomic.Bool
	return m.RegisterFront(func(data *Msgq_tag_data[T]) bool {
		if !disable.Load() && data.Tag == key {
			go func() {
				disable.Store(f(data.Data))
			}()
		}
		return disable.Load()
	})
}

func (m *Msgq) PullSignAsyncOnly(key string, f func(_ struct{}) (disable bool)) (cancel func()) {
	return m.Pull_tag_async_only(key, f)
}

func (m *Msgq) Pull_tag_async[T any](func_map map[string]func(T) (disable bool)) (cancel func()) {
	var disable atomic.Bool
	return m.RegisterFront(func(data *Msgq_tag_data[T]) bool {
		if f, ok := func_map[data.Tag]; !disable.Load() && ok {
			go func() {
				disable.Store(f(data.Data))
			}()
		}
		return disable.Load()
	})
}

func (m *Msgq) PullSignAsync(func_map map[string]func(_ struct{}) (disable bool)) (cancel func()) {
	return m.Pull_tag_async(func_map)
}

type MsgType[T any] struct {
	m *Msgq
}

type MsgType_tag_data[T any] struct {
	Tag  string
	Data *T
}

func (t MsgType_tag_data[T]) String() string {
	return fmt.Sprintf("tag:%v", t.Tag)
}

// to[0]:timeout to wait to[1]:timeout to run
func NewType[T any](to ...time.Duration) *MsgType[T] {
	return &MsgType[T]{m: New(to...)}
}

func (m *MsgType[T]) ClearAll() {
	m.m.ClearAll()
}

func (m *MsgType[T]) TOPanicFunc(f func(any)) {
	m.m.TOPanicFunc(f)
}

// 不能在由PushLock*调用的Pull中以同步方式使用
func (m *MsgType[T]) Push_tag(Tag string, Data T) {
	// if len(m.m.to) > 1 {
	// 	defer m.m.PushingTO(lmt.Sprintf("\nPush_tag(`%v`)", Tag), getCall(1))()
	// }
	m.m.Push(&MsgType_tag_data[T]{
		Tag:  Tag,
		Data: &Data,
	})
}

// 不能在由Push*调用的Pull中以同步方式使用
//
// async类的Pull将会创建协程处理并退出，可能不会按预期工作
func (m *MsgType[T]) PushLock_tag(Tag string, Data T) {
	// if len(m.m.to) > 1 {
	// 	defer m.m.PushingTO(lmt.Sprintf("\nPushLock_tag(`%v`)", Tag), getCall(1))()
	// }
	m.m.PushLock(&MsgType_tag_data[T]{
		Tag:  Tag,
		Data: &Data,
	})
}

func (m *MsgType[T]) Pull_tag_chan(key string, size int, ctx context.Context) (cancel func(), ch <-chan T) {
	c := make(chan T, size)
	ctx, cancelC := context.WithCancel(ctx)
	unreg := m.m.Register(func(data *MsgType_tag_data[T]) bool {
		if data.Tag == key {
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
						c <- *data.Data
						empty = true
					}
				}
			}
		}
		return false
	})
	return func() {
		select {
		case <-ctx.Done():
		default:
			cancelC()
			unreg()
			close(c)
		}
	}, c
}

func (m *MsgType[T]) Pull_tag_only(key string, f func(T) (disable bool)) (cancel func()) {
	return m.m.Register(func(data *MsgType_tag_data[T]) (disable bool) {
		if data.Tag == key {
			return f(*data.Data)
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag(func_map map[string]func(T) (disable bool)) (cancel func()) {
	return m.m.Register(func(data *MsgType_tag_data[T]) (disable bool) {
		if f, ok := func_map[data.Tag]; ok {
			return f(*data.Data)
		}
		return false
	})
}

type RegisterT[T any] struct {
	mq     *Msgq
	cancel atomic.Pointer[func()]
}

func (m *RegisterT[T]) Tag(key string, f func(T) (disable bool)) {
	cancel := m.mq.Register(func(data *MsgType_tag_data[T]) (disable bool) {
		if data.Tag == key {
			disable = f(*data.Data)
		}
		if disable {
			(*m.cancel.Load())()
		}
		return
	})
	{
		var pre *func()
		var cur = func() {
			if pre != nil {
				(*pre)()
			}
			cancel()
		}
		pre = m.cancel.Swap(&cur)
	}
}

func (m *RegisterT[T]) TagAsync(key string, f func(T) (disable bool)) {
	var disabled atomic.Bool
	cancel := m.mq.RegisterFront(func(data *MsgType_tag_data[T]) bool {
		if !disabled.Load() && data.Tag == key {
			go func() {
				disabled.CompareAndSwap(false, f(*data.Data))
			}()
		}
		return disabled.Load()
	})
	{
		var pre *func()
		var cur = func() {
			if pre != nil {
				(*pre)()
			}
			cancel()
		}
		pre = m.cancel.Swap(&cur)
	}
}

func (m *MsgType[T]) Pull_tags(batchTag func(fc *RegisterT[T])) (cancel func()) {
	reg := &RegisterT[T]{mq: m.m}
	batchTag(reg)
	return *reg.cancel.Load()
}

// func return disable will compareAndDelete this func, not remove func_map from msg
func (m *MsgType[T]) Pull_tag_syncmap(func_map psync.MapFunc[string, *func(T) (disable bool)]) (cancel func()) {
	return m.m.Register(func(data *MsgType_tag_data[T]) (disable bool) {
		if f, ok := func_map.Load(data.Tag); ok {
			if disable := (*f)(*data.Data); disable {
				func_map.CompareAndDelete(data.Tag, f)
			}
		}
		return false
	})
}

func (m *MsgType[T]) Pull_tag_async_only(key string, f func(T) (disable bool)) (cancel func()) {
	var disabled atomic.Bool
	return m.m.RegisterFront(func(data *MsgType_tag_data[T]) bool {
		if !disabled.Load() && data.Tag == key {
			go func() {
				if !disabled.Load() {
					disabled.Store(f(*data.Data))
				}
			}()
		}
		return disabled.Load()
	})
}

func (m *MsgType[T]) Pull_tag_async(func_map map[string]func(T) (disable bool)) (cancel func()) {
	var disabled atomic.Bool
	return m.m.RegisterFront(func(data *MsgType_tag_data[T]) bool {
		if !disabled.Load() {
			if f, ok := func_map[data.Tag]; ok {
				go func() {
					if !disabled.Load() {
						disabled.Store(f(*data.Data))
					}
				}()
			}
		}
		return disabled.Load()
	})
}

var pkName = func() string {
	if pc, file, _, ok := runtime.Caller(1); !ok || strings.HasPrefix(file, build.Default.GOROOT) {
		return ""
	} else {
		tmp := runtime.FuncForPC(pc).Name()
		return tmp[:len(tmp)-4] + "("
	}
}()

func getCall() (calls *string) {
	var cs string
	for i := 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, build.Default.GOROOT) {
			break
		} else if pcName := runtime.FuncForPC(pc).Name(); strings.HasPrefix(pcName, pkName) {
			// cs += fmt.Sprintf("\ncall by %s\n\t%s:%d", pcName, file, line)
			continue
		} else {
			cs += fmt.Sprintf("\ncall by %s\n\t%s:%d", pcName, file, line)
		}
	}
	if cs == "" {
		cs += fmt.Sprintln("\ncall by goroutine")
	}
	return &cs
}
