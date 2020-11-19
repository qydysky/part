package part

import (
	// "fmt"
	"sync"
	"time"
	p "github.com/qydysky/part"
)

const push_mute = 25

type msgq struct {
	data msgq_item
	wait_push p.Signal
	push_mute int
	sync.RWMutex
}

type msgq_item struct {
	data interface{}
	sig time.Time
}

func New(want_push_mute ...int) (*msgq) {
	m := new(msgq)
	(*m).wait_push.Init()
	(*m).data.sig = time.Now()
	(*m).push_mute = push_mute
	//太小的禁言时间,将可能出现丢失push的情况
	if len(want_push_mute) != 0 && want_push_mute[0] >= 0{(*m).push_mute = want_push_mute[0]}
	return m
}

func (m *msgq) Push(msg interface{}) {
	m.Lock()
	defer m.Unlock()

	m.wait_push.Done()

	m.data = msgq_item{
		data:msg,
		sig:time.Now(),
	}

	if m.push_mute != 0 {p.Sys().MTimeoutf(m.push_mute)}
}

func (m *msgq) Pull(sig ...time.Time) (interface{},time.Time) {
	if m.wait_push.Islive() || len(sig) != 0 && sig[0] == m.Sig() {
		m.wait_push.Init().Wait()
	}
	//若不提供sig用以识别，将可能出现重复取得的情况

	m.RLock()
	defer m.RUnlock()
	return m.data.data,m.Sig()
}

func (m *msgq) Sig() (time.Time) {
	return m.data.sig
}