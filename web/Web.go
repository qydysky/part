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

	"github.com/dustin/go-humanize"
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

	conf.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, m, c)
	}
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
	path string
	// current net.Conn: conn, ok := r.Context().Value(&WebPath).(net.Conn)
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

type Limits struct {
	g []*limitItem
	l sync.RWMutex
}

func (t *Limits) AddLimitItem(item *limitItem) {
	t.g = append(t.g, item)
}

func (t *Limits) ReachMax(r *http.Request) (isOverflow bool) {
	if len(t.g) == 0 {
		return
	}
	ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	for i := 0; !isOverflow && i < len(t.g); i++ {
		var match bool
		t.g[i].l.RLock()
		for b := 0; !match && b < len(t.g[i].matchfs); b++ {
			switch t.g[i].matchfs[b].k {
			case rcidr:
				match = t.g[i].matchfs[b].f(ip)
			case rreq:
				match = t.g[i].matchfs[b].f(r)
			default:
			}
		}
		if match && t.g[i].available == 0 {
			isOverflow = true
		}
		t.g[i].l.RUnlock()
	}
	return
}

func (t *Limits) AddCount(r *http.Request) (isOverflow bool) {
	if len(t.g) == 0 {
		return
	}
	ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	var matchs []int

	for i := 0; !isOverflow && i < len(t.g); i++ {
		var match bool
		t.g[i].l.RLock()
		for b := 0; !match && b < len(t.g[i].matchfs); b++ {
			switch t.g[i].matchfs[b].k {
			case rcidr:
				match = t.g[i].matchfs[b].f(ip)
			case rreq:
				match = t.g[i].matchfs[b].f(r)
			default:
			}
		}
		if match {
			if t.g[i].available == 0 {
				isOverflow = true
			} else {
				matchs = append(matchs, i)
			}
		}
		t.g[i].l.RUnlock()
	}
	if !isOverflow && len(matchs) != 0 {
		t.l.Lock()
		for i := 0; !isOverflow && i < len(matchs); i++ {
			t.g[matchs[i]].l.RLock()
			if t.g[matchs[i]].available == 0 {
				isOverflow = true
			}
			t.g[matchs[i]].l.RUnlock()
		}
		for i := 0; !isOverflow && i < len(matchs); i++ {
			t.g[matchs[i]].l.Lock()
			t.g[matchs[i]].available -= 1
			t.g[matchs[i]].l.Unlock()
		}
		t.l.Unlock()
		if !isOverflow {
			go func() {
				<-r.Context().Done()
				t.l.Lock()
				for i := 0; i < len(matchs); i++ {
					t.g[matchs[i]].l.Lock()
					t.g[matchs[i]].available += 1
					t.g[matchs[i]].l.Unlock()
				}
				t.l.Unlock()
			}()
		}
	}
	return
}

const (
	rcidr = iota
	rreq
)

type limitItem struct {
	matchfs   []matchFunc
	available int
	l         sync.RWMutex
}

type matchFunc struct {
	k int
	f func(any) (match bool)
}

func NewLimitItem(max int) *limitItem {
	return &limitItem{
		available: max,
	}
}

func (t *limitItem) Cidr(cidr string) *limitItem {
	if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
		panic(err)
	} else {
		t.matchfs = append(t.matchfs, matchFunc{
			rcidr,
			func(a any) (match bool) {
				return cidrx.Contains(a.(net.IP))
			},
		})
	}
	return t
}

func (t *limitItem) Request(matchf func(req *http.Request) (match bool)) *limitItem {
	t.matchfs = append(t.matchfs, matchFunc{
		rreq,
		func(a any) (match bool) {
			return matchf(a.(*http.Request))
		},
	})
	return t
}

type Cache struct {
	g   psync.MapExceeded[string, []byte]
	gcL atomic.Int64
}

type cacheRes struct {
	headerf      func() http.Header
	writef       func([]byte) (int, error)
	writeHeaderf func(statusCode int)
}

func (t cacheRes) Header() http.Header {
	return t.headerf()
}
func (t cacheRes) Write(b []byte) (int, error) {
	return t.writef(b)
}
func (t cacheRes) WriteHeader(statusCode int) {
	t.writeHeaderf(statusCode)
}

func (t *Cache) IsCache(key string) (data *[]byte, isCache bool) {
	return t.g.Load(key)
}

func (t *Cache) Cache(key string, aliveDur time.Duration, w http.ResponseWriter) http.ResponseWriter {
	var (
		res    cacheRes
		called atomic.Bool
	)
	res.headerf = w.Header
	res.writeHeaderf = w.WriteHeader
	res.writef = func(b []byte) (int, error) {
		if called.CompareAndSwap(false, true) {
			if len(b) < humanize.MByte {
				t.g.Store(key, &b, aliveDur)
			}
		} else {
			panic("Cache Write called")
		}
		return w.Write(b)
	}
	if s := int64(t.g.Len()); s > 10 && t.gcL.Load() <= s {
		t.gcL.Store(s * 2)
		t.g.GC()
	}
	return res
}

type withflush struct {
	raw http.ResponseWriter
}

func (t withflush) Header() http.Header {
	if t.raw != nil {
		return t.raw.Header()
	}
	return make(http.Header)
}
func (t withflush) Write(b []byte) (i int, e error) {
	if t.raw != nil {
		i, e = t.Write(b)
		if e != nil {
			return
		}
		if flusher, ok := t.raw.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	return
}
func (t withflush) WriteHeader(i int) {
	if t.raw != nil {
		t.raw.WriteHeader(i)
	}
}

func WithFlush(w http.ResponseWriter) http.ResponseWriter {
	return withflush{w}
}

func WithStatusCode(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(http.StatusText(code)))
}

func IsMethod(r *http.Request, method ...string) bool {
	for i := 0; i < len(method); i++ {
		if r.Method == method[i] {
			return true
		}
	}
	return false
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
