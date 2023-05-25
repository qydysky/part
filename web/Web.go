package part

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	psync "github.com/qydysky/part/sync"
	sys "github.com/qydysky/part/sys"
)

type Web struct {
	Server *http.Server
	mux    *http.ServeMux
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

func (t *Web) Handle(path_func map[string]func(http.ResponseWriter, *http.Request)) {
	for k, v := range path_func {
		t.mux.HandleFunc(k, v)
	}
}

func (t *Web) Shutdown() {
	t.Server.Shutdown(context.Background())
}

type WebSync struct {
	Server *http.Server
	mux    *http.ServeMux
	wrs    *WebPath
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

func (t *WebSync) Shutdown() {
	t.Server.Shutdown(context.Background())
}

type WebPath struct {
	path  string
	f     func(w http.ResponseWriter, r *http.Request)
	sameP *WebPath
	next  *WebPath
	sync.RWMutex
}

func (t *WebPath) Load(path string) (func(w http.ResponseWriter, r *http.Request), bool) {
	t.RLock()
	defer t.RUnlock()
	if t.path == path || t.f == nil {
		// 操作本节点
		return t.f, true
	} else if lp, ltp := len(path), len(t.path); lp > ltp && path[:ltp] == t.path && (path[ltp] == '/' || t.path[ltp-1] == '/') {
		// 操作sameP节点
		if t.sameP != nil {
			if f, ok := t.sameP.Load(path); ok {
				return f, true
			}
		}
		if t.path[ltp-1] == '/' {
			return t.f, true
		} else {
			return nil, false
		}
	} else if lp < ltp && t.path[:lp] == path && (path[lp-1] == '/' || t.path[lp] == '/') {
		// 操作sameP节点
		return nil, false
	} else {
		// 操作next节点
		if t.next != nil {
			if f, ok := t.next.Load(path); ok {
				return f, true
			}
		}
		return nil, false
	}
}

func (t *WebPath) Store(path string, f func(w http.ResponseWriter, r *http.Request)) {
	t.Lock()
	defer t.Unlock()
	if t.path == path || t.f == nil {
		// 操作本节点
		t.path = path
		t.f = f
	} else if len(path) > len(t.path) && path[:len(t.path)] == t.path && (path[len(t.path)-1] == '/' || t.path[len(t.path)-1] == '/') {
		// 操作sameP节点
		if t.sameP != nil {
			t.sameP.Store(path, f)
		} else {
			t.sameP = &WebPath{
				path: path,
				f:    f,
			}
		}
	} else if len(path) < len(t.path) && t.path[:len(path)] == path && (path[len(path)-1] == '/' || t.path[len(path)-1] == '/') {
		// 操作sameP节点
		tmp := WebPath{path: t.path, f: t.f, sameP: t.sameP, next: t.next}
		t.path = path
		t.f = f
		t.sameP = &tmp
		t.next = nil
	} else {
		// 操作next节点
		if t.next != nil {
			t.next.Store(path, f)
		} else {
			t.next = &WebPath{
				path: path,
				f:    f,
			}
		}
	}
}

type CountLimits struct {
	g []countLimit
	l sync.RWMutex
}

type countLimit struct {
	cidr      *net.IPNet
	available int
}

func (t *CountLimits) SetMaxCount(cidr string, max int) {
	if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
		panic(err)
	} else {
		t.g = append(t.g, countLimit{cidrx, max})
	}
}

func (t *CountLimits) ReachMax(r *http.Request) (isOverflow bool) {
	if len(t.g) == 0 {
		return
	}
	ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	t.l.RLock()
	defer t.l.RUnlock()
	for i := 0; i < len(t.g); i++ {
		if !t.g[i].cidr.Contains(ip) {
			continue
		}
		if t.g[i].available == 0 {
			isOverflow = true
			break
		}
	}
	return
}

func (t *CountLimits) AddCount(r *http.Request) (isOverflow bool) {
	if len(t.g) == 0 {
		return
	}
	ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	t.l.Lock()
	defer t.l.Unlock()
	var match []int
	for i := 0; i < len(t.g); i++ {
		if !t.g[i].cidr.Contains(ip) {
			continue
		}
		if t.g[i].available == 0 {
			isOverflow = true
			return
		}
		match = append(match, i)
	}
	for i := 0; i < len(match); i++ {
		t.g[match[i]].available -= 1
	}
	go func() {
		<-r.Context().Done()
		t.l.Lock()
		defer t.l.Unlock()
		for i := 0; i < len(match); i++ {
			t.g[match[i]].available += 1
		}
	}()
	return
}

type Cache struct {
	g   psync.MapExceeded[string, []byte]
	gcL atomic.Int64
}

func (t *Cache) IsCache(key string) (res *[]byte, isCache bool) {
	return t.g.Load(key)
}

func (t *Cache) Store(key string, aliveDur time.Duration, data *[]byte) {
	t.g.Store(key, data, aliveDur)
	if s := int64(t.g.Len()); s > 10 && t.gcL.Load() <= s {
		t.gcL.Store(s * 2)
		t.g.GC()
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
		`/exit`: func(_ http.ResponseWriter, _ *http.Request) {
			s.Server.Shutdown(context.Background())
		},
	})
	return s
}
