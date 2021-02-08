package part

import (
	"sync"
	"time"
	"runtime"
	"container/list"
)

type Msgq struct {
	data_list *list.List
	wait_push chan struct{}
	max_data_mun int
	ticker *time.Ticker
	sig uint64
	sync.RWMutex
}

type Msgq_item struct {
	data interface{}
	sig uint64
}

type FuncMap map[string]func(interface{})(bool)

func New(want_max_data_mun int) (*Msgq) {
	m := new(Msgq)
	(*m).wait_push = make(chan struct{},10)
	(*m).data_list = list.New()
	(*m).max_data_mun = want_max_data_mun
	(*m).ticker = time.NewTicker(time.Duration(25)*time.Millisecond)
	return m
}

func (m *Msgq) Push(msg interface{}) {
	m.Lock()
	defer m.Unlock()
	m.data_list.PushBack(Msgq_item{
		data:msg,
		sig:m.get_sig(),
	})
	if m.data_list.Len() > m.max_data_mun {m.data_list.Remove(m.data_list.Front())}

	var pull_num int
	for len(m.wait_push) == 0 {
		pull_num += 1
		m.wait_push <- struct{}{}
	}
	if pull_num < 1 {<- m.ticker.C}
	runtime.Gosched()
	select {
	case <- m.wait_push:
	case <- m.ticker.C:
	}
}

func (m *Msgq) Pull(old_sig uint64) (data interface{},sig uint64) {
	for old_sig == m.Sig() {
		select {
		case <- m.wait_push:
		case <- m.ticker.C:
		}
	}
	m.RLock()
	defer m.RUnlock()

	if int(m.Sig() - old_sig) > m.max_data_mun {return nil,m.Sig()}

	for el := m.data_list.Front();el != nil;el = el.Next() {
		if old_sig < el.Value.(Msgq_item).sig {
			data = el.Value.(Msgq_item).data
			sig = el.Value.(Msgq_item).sig
			return
		}
	}
	return
}

func (m *Msgq) Sig() (sig uint64) {
	if el := m.data_list.Back();el == nil {
		sig = 0
	} else {
		sig = el.Value.(Msgq_item).sig
	}
	return
}

func (m *Msgq) get_sig() (sig uint64) {
	m.sig += 1
	return m.sig
}

type Msgq_tag_data struct {
	Tag string
	Data interface{}
}

func (m *Msgq) Push_tag(Tag string,Data interface{}) {
	m.Push(Msgq_tag_data{
		Tag:Tag,
		Data:Data,
	})
}

func (m *Msgq) Pull_tag(func_map map[string]func(interface{})(bool)) {
	go func(){
		var (
			sig = m.Sig()
			data interface{}
		)
		for {
			data,sig = m.Pull(sig)
			if d,ok := data.(Msgq_tag_data);!ok{
				if f,ok := func_map[`Error`];ok{
					if f(d.Data) {break}
				}
			} else {
				if f,ok := func_map[d.Tag];ok{
					if f(d.Data) {break}
				}
			}
		}
	}()
}