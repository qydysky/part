package part

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	sys "github.com/qydysky/part/sys"
)

type Web struct {
	Server *http.Server
	mux    *http.ServeMux
}

type WebSync struct {
	Server *http.Server
	mux    *http.ServeMux
	wrs    *WebPath
}

type WebPath struct {
	path  string
	f     func(w http.ResponseWriter, r *http.Request)
	sameP *WebPath
	next  *WebPath
	l     sync.RWMutex
}

func (t *WebPath) Load(path string) (func(w http.ResponseWriter, r *http.Request), bool) {
	fmt.Println("l", t.path, path)
	t.l.RLock()
	if t.path == "" {
		t.l.RUnlock()
		return nil, false
	} else if t.path == path {
		t.l.RUnlock()
		return t.f, true
	} else if len(path) > len(t.path) && path[:len(t.path)] == t.path {
		if t.path == "/" || path[len(t.path)] == '/' {
			fmt.Println("-")
			if t.sameP != nil {
				if f, ok := t.sameP.Load(path); ok {
					t.l.RUnlock()
					return f, true
				} else {
					t.l.RUnlock()
					return t.f, true
				}
			} else {
				t.l.RUnlock()
				return t.f, true
			}
		} else {
			if t.next != nil {
				t.l.RUnlock()
				return t.next.Load(path)
			} else {
				t.l.RUnlock()
				return nil, false
			}
		}
	} else if t.next != nil {
		t.l.RUnlock()
		return t.next.Load(path)
	} else {
		t.l.RUnlock()
		return nil, false
	}
}

func (t *WebPath) Store(path string, f func(w http.ResponseWriter, r *http.Request)) {
	t.l.RLock()
	if t.path == path || t.path == "" {
		t.l.RUnlock()
		t.l.Lock()
		t.path = path
		t.f = f
		t.l.Unlock()
	} else if len(path) > len(t.path) && path[:len(t.path)] == t.path {
		if path[len(t.path)-1] == '/' {
			if t.sameP != nil {
				t.l.RUnlock()
				t.sameP.Store(path, f)
			} else {
				t.l.RUnlock()
				t.l.Lock()
				t.sameP = &WebPath{
					path: path,
					f:    f,
				}
				t.l.Unlock()
			}
		} else {
			if t.next != nil {
				t.l.RUnlock()
				t.l.Lock()
				tmp := WebPath{path: t.path, f: t.f, sameP: t.sameP, next: t.next}
				t.path = path
				t.f = f
				t.next = &tmp
				t.l.Unlock()
			} else {
				t.l.RUnlock()
				t.l.Lock()
				t.next = &WebPath{
					path: path,
					f:    f,
				}
				t.l.Unlock()
			}
		}
	} else if len(path) < len(t.path) && t.path[:len(path)] == path {
		t.l.RUnlock()
		t.l.Lock()
		tmp := WebPath{path: t.path, f: t.f, sameP: t.sameP, next: t.next}
		t.path = path
		t.f = f
		t.sameP = &tmp
		t.l.Unlock()
	} else if t.next != nil {
		t.l.RUnlock()
		t.next.Store(path, f)
	} else {
		t.l.RUnlock()
		t.l.Lock()
		t.next = &WebPath{
			path: path,
			f:    f,
		}
		t.l.Unlock()
	}
}

func New(conf *http.Server) (o *Web) {

	o = new(Web)

	o.Server = conf

	if o.Server.Handler == nil {
		o.mux = http.NewServeMux()
		o.Server.Handler = o.mux
	}

	go o.Server.ListenAndServe()

	return
}

func NewSyncMap(conf *http.Server, m *WebPath) (o *WebSync) {

	o = new(WebSync)

	o.Server = conf
	o.wrs = m

	if o.Server.Handler == nil {
		o.mux = http.NewServeMux()
		o.Server.Handler = o.mux
	}

	go o.Server.ListenAndServe()

	o.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if f, ok := o.wrs.Load(r.URL.Path); ok {
			f(w, r)
		}
	})

	return
}

func (t *Web) Handle(path_func map[string]func(http.ResponseWriter, *http.Request)) {
	for k, v := range path_func {
		t.mux.HandleFunc(k, v)
	}
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
