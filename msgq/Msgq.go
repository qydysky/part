package part

import (
	"sync"
	"container/list"
)

type msgq struct {
	data_list *list.List
	wait_push chan bool
	max_data_mun int
	sig uint64
	sync.RWMutex
}

type msgq_item struct {
	data interface{}
	sig uint64
}

func New(want_max_data_mun int) (*msgq) {
	m := new(msgq)
	(*m).wait_push = make(chan bool,100)
	(*m).data_list = list.New()
	(*m).max_data_mun = want_max_data_mun
	return m
}

func (m *msgq) Push(msg interface{}) {
	m.Lock()
	defer m.Unlock()
	m.data_list.PushBack(msgq_item{
		data:msg,
		sig:m.get_sig(),
	})
	if m.data_list.Len() > m.max_data_mun {m.data_list.Remove(m.data_list.Front())}
	for len(m.wait_push) == 0 {m.wait_push <- true}
	<- m.wait_push
}

func (m *msgq) Pull(old_sig uint64) (data interface{},sig uint64) {
	if old_sig == m.Sig() || m.data_list.Len() == 0 {<- m.wait_push}
	m.RLock()
	defer m.RUnlock()
	for el := m.data_list.Front();el != nil;el = el.Next() {
		if old_sig < el.Value.(msgq_item).sig {
			data = el.Value.(msgq_item).data
			sig = el.Value.(msgq_item).sig
			break
		}
	}
	return
}

func (m *msgq) Sig() (sig uint64) {
	if el := m.data_list.Back();el == nil {
		sig = m.get_sig()
	} else {
		sig = el.Value.(msgq_item).sig
	}
	return
}

func (m *msgq) get_sig() (sig uint64) {
	m.sig += 1
	return m.sig
}