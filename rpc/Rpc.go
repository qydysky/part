package part

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"net/rpc"
	"sync"

	pp "github.com/qydysky/part/pool"
	web "github.com/qydysky/part/web"
)

var (
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
}

// func NewGob(ptr any) *Gob {
// 	t := new(Gob)
// 	var buf bytes.Buffer
// 	t.Err = gob.NewEncoder(&buf).Encode(ptr)
// 	t.Data = buf.Bytes()
// 	return t
// }

func (t *Gob) encode(ptr any) (e error) {
	var buf bytes.Buffer
	e = gob.NewEncoder(&buf).Encode(ptr)
	t.Data = buf.Bytes()
	return
}

func (t *Gob) decode(ptr any) (e error) {
	return gob.NewDecoder(bytes.NewReader(t.Data)).Decode(ptr)
}

type GobCoder struct {
	g   *Gob
	buf bytes.Buffer
	enc *gob.Encoder
	dnc *gob.Decoder
}

var ErrGobCoderLockFail = errors.New(`ErrGobCoderLockFail`)

func NewGobCoder(g *Gob) *GobCoder {
	t := &GobCoder{g: g}
	t.enc = gob.NewEncoder(&t.buf)
	t.dnc = gob.NewDecoder(&t.buf)
	return t
}

func (t *GobCoder) encode(ptr any) (e error) {
	t.buf.Reset()
	e = t.enc.Encode(ptr)
	t.g.Data = t.buf.Bytes()
	return
}

func (t *GobCoder) decode(ptr any) (e error) {
	e = t.dnc.Decode(ptr)
	t.buf.Reset()
	return
}

type GobCoders struct {
	gs sync.Map
}

func (t *GobCoders) New() *GobCoder {
	return t.Get(new(Gob))
}

func (t *GobCoders) Get(g *Gob) *GobCoder {
	if c, ok := t.gs.Load(g); ok {
		return c.(*GobCoder)
	} else {
		c, _ = t.gs.LoadOrStore(g, NewGobCoder(g))
		return c.(*GobCoder)
	}
}

func (t *GobCoders) Del(g *Gob) {
	t.gs.Delete(g)
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
	ser := new(Server)
	webSync := web.NewSyncMap(&http.Server{
		Addr: host,
	}, &ser.webP)
	ser.Shutdown = webSync.Shutdown
	return ser
}

func Register[T, E any](t *Server, path string, deal func(it *T, ot *E) error) error {
	s := rpc.NewServer()
	if e := s.Register(newDealGob(func(i, o *Gob) error {
		var it T
		var ot E
		if e := i.decode(&it); e != nil {
			return errors.Join(ErrSerDecode, e)
		}
		if e := deal(&it, &ot); e != nil {
			return errors.Join(ErrSerDeal, e)
		}
		if e := o.encode(&ot); e != nil {
			return errors.Join(ErrSerEncode, e)
		}
		return nil
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
	t := new(Gob)
	if e := t.encode(it); e != nil {
		return errors.Join(ErrCliEncode, e)
	} else {
		if c, e := rpc.DialHTTPPath("tcp", host, path); e != nil {
			return errors.Join(ErrDial, e)
		} else {
			call := <-c.Go("DealGob.Deal", t, t, make(chan *rpc.Call, 1)).Done
			if call.Error != nil {
				return errors.Join(ErrCliDeal, call.Error)
			}
			if e := t.decode(ot); e != nil {
				return errors.Join(ErrCliDecode, e)
			}
			return nil
		}
	}
}

func CallReuse[T, E any](host, path string, poolSize int) func(it *T, ot *E) error {
	var gobs = GobCoders{}
	gobPool := pp.New(pp.PoolFunc[GobCoder]{
		New: gobs.New,
	}, poolSize)
	c, ce := rpc.DialHTTPPath("tcp", host, path)
	return func(it *T, ot *E) (e error) {
		if ce != nil {
			return errors.Join(ErrDial, ce)
		}
		t := gobPool.Get()
		defer gobPool.Put(t)
		if e := t.encode(it); e != nil {
			return errors.Join(ErrCliEncode, e)
		}
		if e := c.Call("DealGob.Deal", t.g, t.g); e != nil {
			return errors.Join(ErrCliDeal, e)
		}
		if e := t.decode(ot); e != nil {
			return errors.Join(ErrCliDecode, e)
		}
		return nil
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
// 	return Call(&info, new(struct{}), regHost, regPath)
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
// 			_ = Call(it, new(struct{}), hosts[0], path)
// 			hosts = hosts[1:]
// 		}
// 	}
// }
