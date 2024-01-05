package part

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"net/rpc"

	web "github.com/qydysky/part/web"
)

var (
	ErrSerGob    = errors.New("ErrSerGob")
	ErrSerDecode = errors.New("ErrSerDecode")
	ErrCliDecode = errors.New("ErrCliDecode")
	ErrSerDeal   = errors.New("ErrSerDeal")
	ErrCliDeal   = errors.New("ErrCliDeal")
	ErrCliEncode = errors.New("ErrCliEncode")
	ErrSerEncode = errors.New("ErrSerEncode")
	ErrRegister  = errors.New("ErrRegister")
	ErrDial      = errors.New("ErrDial")
)

type Gob struct {
	Data []byte
	Err  error
}

// func NewGob(ptr any) *Gob {
// 	t := &Gob{}
// 	var buf bytes.Buffer
// 	t.Err = gob.NewEncoder(&buf).Encode(ptr)
// 	t.Data = buf.Bytes()
// 	return t
// }

func (t *Gob) encode(ptr any) *Gob {
	var buf bytes.Buffer
	t.Err = gob.NewEncoder(&buf).Encode(ptr)
	t.Data = buf.Bytes()
	return t
}

func (t *Gob) decode(ptr any) *Gob {
	t.Err = gob.NewDecoder(bytes.NewReader(t.Data)).Decode(ptr)
	return t
}

// func (t *Gob) RpcDeal(host, path string) *Gob {
// 	if t.Err == nil {
// 		if c, e := rpc.DialHTTPPath("tcp", host, path); e != nil {
// 			t.Err = e
// 		} else {
// 			call := <-c.Go("DealGob.Deal", t, t, make(chan *rpc.Call, 1)).Done
// 			if call.Error != nil {
// 				t.Err = call.Error
// 			}
// 		}
// 	}
// 	return t
// }

type DealGob struct {
	deal func(*Gob, *Gob) error
}

func newDealGob(deal func(i *Gob, o *Gob) error) *DealGob {
	return &DealGob{deal}
}

func (t *DealGob) Deal(i *Gob, o *Gob) error {
	return t.deal(i, o)
}

type Server struct {
	webP     web.WebPath
	Shutdown func(ctx ...context.Context)
}

func NewServer(host string) *Server {
	ser := &Server{}
	webSync := web.NewSyncMap(&http.Server{
		Addr: host,
	}, &ser.webP)
	ser.Shutdown = webSync.Shutdown
	return ser
}

func Register[T, E any](t *Server, path string, deal func(it *T, ot *E) error) error {
	s := rpc.NewServer()
	if e := s.Register(newDealGob(func(i, o *Gob) error {
		if i.Err != nil {
			return errors.Join(ErrSerGob, i.Err)
		} else {
			var it T
			var ot E
			if e := i.decode(&it).Err; e != nil {
				return errors.Join(ErrSerDecode, e)
			}
			if e := deal(&it, &ot); e != nil {
				return errors.Join(ErrSerDeal, e)
			}
			if e := o.encode(&ot).Err; e != nil {
				return errors.Join(ErrSerEncode, e)
			}
			return nil
		}
	})); e != nil {
		return errors.Join(ErrRegister, e)
	}
	t.webP.Store(path, func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})
	return nil
}

func UnRegister(t *Server, path string) {
	t.webP.Store(path, nil)
}

func Call[T, E any](host, path string, it *T, ot *E) error {
	var buf bytes.Buffer
	if e := gob.NewEncoder(&buf).Encode(it); e != nil {
		return errors.Join(ErrCliEncode, e)
	} else {
		if c, e := rpc.DialHTTPPath("tcp", host, path); e != nil {
			return errors.Join(ErrDial, e)
		} else {
			t := &Gob{Data: buf.Bytes()}
			call := <-c.Go("DealGob.Deal", t, t, make(chan *rpc.Call, 1)).Done
			if call.Error != nil {
				return errors.Join(ErrCliDeal, call.Error)
			}
			if t.Err != nil {
				return errors.Join(ErrSerGob, t.Err)
			}
			if e := gob.NewDecoder(bytes.NewReader(t.Data)).Decode(ot); e != nil {
				return errors.Join(ErrCliDecode, e)
			}
			return nil
		}
	}
}

// var (
// 	ErrRegUnKnowMethod = errors.New("ErrRegUnKnowMethod")
// )

// type RegisterSer struct {
// 	m        map[string]*registerItem
// 	Shutdown func(ctx ...context.Context)
// 	sync.RWMutex
// }

// type RegisterSerHost struct {
// 	Act  string
// 	Host string
// 	Path string
// }

// type registerItem struct {
// 	hosts map[string]struct{}
// 	sync.RWMutex
// }

// func NewRegisterSer(host, path string) (ser *RegisterSer, e error) {
// 	ser = &RegisterSer{m: make(map[string]*registerItem)}
// 	s := NewServer(host)
// 	ser.Shutdown = s.Shutdown
// 	e = Register(s, path, func(it *RegisterSerHost, ot *struct{}) error {
// 		if it.Act == "add" {
// 			ser.RLock()
// 			item, ok := ser.m[it.Path]
// 			ser.RUnlock()
// 			if ok {
// 				item.Lock()
// 				item.hosts[it.Host] = struct{}{}
// 				item.Unlock()
// 			} else {
// 				ser.Lock()
// 				l := make(map[string]struct{})
// 				l[it.Host] = struct{}{}
// 				ser.m[it.Path] = &registerItem{
// 					hosts: l,
// 				}
// 				ser.Unlock()
// 			}
// 		} else if it.Act == "del" {
// 			ser.RLock()
// 			item, ok := ser.m[it.Path]
// 			ser.RUnlock()
// 			if ok {
// 				item.Lock()
// 				delete(item.hosts, it.Host)
// 				item.Unlock()
// 			}
// 		} else {
// 			return ErrRegUnKnowMethod
// 		}
// 		return nil
// 	})
// 	return
// }

// func RegisterSerReg(regHost, regPath string, info RegisterSerHost) error {
// 	return Call(&info, &struct{}{}, regHost, regPath)
// }

// func RegisterSerCall[T any](ser *RegisterSer, it *T, path string) {
// 	ser.RLock()
// 	item, ok := ser.m[path]
// 	ser.RUnlock()
// 	if ok {
// 		var hosts []string
// 		item.RLock()
// 		for host, _ := range item.hosts {
// 			hosts = append(hosts, host)
// 		}
// 		item.RUnlock()

// 		for len(hosts) > 0 {
// 			_ = Call(it, &struct{}{}, hosts[0], path)
// 			hosts = hosts[1:]
// 		}
// 	}
// }
