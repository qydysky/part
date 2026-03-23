package server

import (
	"context"
	"net/http"
	"unsafe"

	pio "github.com/qydysky/part/io"
	pmq "github.com/qydysky/part/msgq"
	pp "github.com/qydysky/part/pool"
	pw "github.com/qydysky/part/web"
)

type Server struct {
	mq       *pmq.MsgType[*Umsg]
	connPool pp.PoolBlockI[Umsg]
}

func NewServer() (s *Server) {
	s = new(Server)
	s.mq = pmq.NewType[*Umsg]()
	s.connPool = pp.NewPoolBlock(func() *Umsg {
		return &Umsg{
			Id:   uintptr(unsafe.Pointer(new(int))),
			Key:  make([]byte, 8),
			Data: make([]byte, 1024),
		}
	})
	return
}

func (t *Server) MQ() *pmq.MsgType[*Umsg] {
	return t.mq
}

// 用于httpSer.Handle里。接入和断开管道各收到一次消息
//
// <- chan *Umsg
//
// 0. 发送到t.mq `init`
//
// 1. 尝试开始sse连接，如有错误，发送到t.mq `error`
//
// 2. 监听t.mq `send`，并发送
//
// 2. 监听t.mq `close`，控制连接断开
//
// 2. 将接收到的数据发送到t.mq `recv`
//
// 2. 如有错误，发送到t.mq `error`
//
// 3. 连接结束，发送到t.mq `fin`
//
// <- chan *Umsg
func (t *Server) Handle(w http.ResponseWriter, r *http.Request) <-chan *Umsg {
	umsg := t.connPool.Get().ReSet()

	umsg.Data, _ = pio.ReadAll(r.Body, umsg.Data)

	ch := make(chan *Umsg, 3)
	go func() {
		defer t.connPool.Put(umsg)
		ch <- umsg
		defer func() {
			ch <- umsg
		}()

		t.mq.Push_tag(`init`, umsg)
		defer t.mq.Push_tag(`fin`, umsg)

		w = pw.WithFlush(w)
		w.Header().Set(`Connection`, `keep-alive`)
		w.Header().Set(`Transfer-Encoding`, `chunked`)
		w.Header().Set(`Content-Type`, `text/event-stream`)
		w.Header().Set(`Cache-Control`, `no-cache`)

		defer t.mq.Pull_tag_only(`send`, func(u *Umsg) (disable bool) {
			if u.Id != 0 && u.Id != umsg.Id {
				return false
			}
			if len(u.Key) != 0 {
				if _, u.Err = w.Write(u.Key); u.Err != nil {
				} else if _, u.Err = w.Write([]byte{':', ' '}); u.Err != nil {
				} else if _, u.Err = w.Write(u.Data); u.Err != nil {
				} else if _, u.Err = w.Write([]byte{'\n'}); u.Err != nil {
				} else {
				}
			} else if _, u.Err = w.Write([]byte{'\n'}); u.Err != nil {
			} else {
			}
			if u.Err != nil {
				t.mq.Push_tag(`error`, u)
				return true
			} else {
				return false
			}
		})()

		cancle, ch := t.mq.Pull_tag_chan(`close`, 2, context.Background())
		defer cancle()

		t.mq.Push_tag(`recv`, umsg)

		for stop := false; !stop; {
			select {
			case uid := <-ch:
				stop = uid.Id == 0 || uid.Id == umsg.Id
			case <-r.Context().Done():
				stop = true
			}
		}
	}()

	return ch
}

type Umsg struct {
	Id   uintptr
	Err  error
	Key  []byte
	Data []byte
}

func (t *Umsg) ReSet() *Umsg {
	t.Key = t.Key[:0]
	t.Data = t.Data[:0]
	t.Err = nil
	return t
}

func (t *Umsg) Set(key, data []byte) *Umsg {
	t.Key = key
	t.Data = append(t.Data[:0], data...)
	return t
}

func (t *Umsg) End() *Umsg {
	t.Key = t.Key[:0]
	return t
}
