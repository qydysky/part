package part

import (
	"net/http"
	"net/rpc"

	web "github.com/qydysky/part/web"
)

type Gob struct {
	Key  string
	Data any
}

type DealGob struct {
	deal func(*Gob, *Gob) error
}

func newDealGob(deal func(i *Gob, o *Gob) error) *DealGob {
	return &DealGob{deal}
}

func (t *DealGob) Deal(i *Gob, o *Gob) error {
	return t.deal(i, o)
}

type Pob struct {
	Host string `json:"host"`
	Path string `json:"path"`
	s    *rpc.Server
	c    *rpc.Client
}

func (t *Pob) Server(deal func(i, o *Gob) error) (shutdown func(), err error) {
	var path web.WebPath
	webSync := web.NewSyncMap(&http.Server{
		Addr: t.Host,
	}, &path)
	shutdown = webSync.Shutdown

	t.s = rpc.NewServer()
	if e := t.s.Register(newDealGob(deal)); e != nil {
		err = e
		return
	}

	path.Store(t.Path, func(w http.ResponseWriter, r *http.Request) {
		t.s.ServeHTTP(w, r)
	})
	return
}

func (t *Pob) Client() (pobClient *PobClient, err error) {
	t.c, err = rpc.DialHTTPPath("tcp", t.Host, t.Path)
	pobClient = &PobClient{t.c}
	return
}

type PobClient struct {
	c *rpc.Client
}

func (t *PobClient) Close() {
	t.c.Close()
}

func (t *PobClient) CallIO(i, o *Gob) (err error) {
	return t.c.Call("DealGob.Deal", i, o)
}

func (t *PobClient) GoIO(i, o *Gob, done chan *rpc.Call) *rpc.Call {
	return t.c.Go("DealGob.Deal", i, o, done)
}

func (t *PobClient) Call(g *Gob) (err error) {
	return t.c.Call("DealGob.Deal", g, g)
}

func (t *PobClient) Go(g *Gob, done chan *rpc.Call) *rpc.Call {
	return t.c.Go("DealGob.Deal", g, g, done)
}
