package part

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	sys "github.com/qydysky/part/sys"
)

type Web struct {
	Server *http.Server
	mux    *http.ServeMux
	wrs    sync.Map
	mode   string
}

func New(conf *http.Server) (o *Web) {

	o = new(Web)

	o.mode = "simple"
	o.Server = conf

	if o.Server.Handler == nil {
		o.mux = http.NewServeMux()
		o.Server.Handler = o.mux
	}

	go o.Server.ListenAndServe()

	return
}

func NewSync(conf *http.Server) (o *Web) {

	o = new(Web)

	o.mode = "sync"
	o.Server = conf

	if o.Server.Handler == nil {
		o.mux = http.NewServeMux()
		o.Server.Handler = o.mux
	}

	go o.Server.ListenAndServe()

	o.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if wr, ok := o.wrs.Load(r.URL.Path); ok {
			if f, ok := wr.(func(http.ResponseWriter, *http.Request)); ok {
				f(w, r)
			}
		}
	})

	return
}

func NewSyncMap(conf *http.Server, m *sync.Map) (o *Web) {

	o = new(Web)

	o.mode = "syncmap"
	o.Server = conf

	if o.Server.Handler == nil {
		o.mux = http.NewServeMux()
		o.Server.Handler = o.mux
	}

	go o.Server.ListenAndServe()

	o.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if wr, ok := m.Load(r.URL.Path); ok {
			if f, ok := wr.(func(http.ResponseWriter, *http.Request)); ok {
				r.URL.Path = "/"
				f(w, r)
			}
		}
	})

	return
}

func (t *Web) Handle(path_func map[string]func(http.ResponseWriter, *http.Request)) {
	if t.mode != "simple" {
		panic("必须是New创建的")
	}
	for k, v := range path_func {
		t.mux.HandleFunc(k, v)
	}
}

func (t *Web) HandleSync(path string, path_func func(http.ResponseWriter, *http.Request)) {
	if t.mode != "sync" {
		panic("必须是NewSync创建的")
	}
	t.wrs.Store(path, path_func)
}

func Easy_boot() *Web {
	s := New(&http.Server{
		Addr:         "127.0.0.1:" + strconv.Itoa(sys.Sys().GetFreePort()),
		WriteTimeout: time.Second * time.Duration(10),
	})
	s.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/`: func(w http.ResponseWriter, r *http.Request) {
			var path string = r.URL.Path[1:]
			if path == `` {
				path = `index.html`
			}
			http.ServeFile(w, r, path)
		},
		`/exit`: func(w http.ResponseWriter, r *http.Request) {
			s.Server.Shutdown(context.Background())
		},
	})
	return s
}
