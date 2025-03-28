package part

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	pio "github.com/qydysky/part/io"
	psync "github.com/qydysky/part/sync"
	sys "github.com/qydysky/part/sys"
)

type Web struct {
	Server *http.Server
	// mux    *http.ServeMux
}

func New(conf *http.Server) (o *Web) {

	o = new(Web)

	o.Server = conf

	// if o.Server.Handler == nil {
	// o.mux = http.NewServeMux()
	// o.Server.Handler = o.mux
	// }

	go func() {
		_ = o.Server.ListenAndServe()
	}()

	return
}

func (t *Web) Handle(path_func map[string]func(http.ResponseWriter, *http.Request)) {
	t.Server.Handler = NewHandler(func(path string) (func(w http.ResponseWriter, r *http.Request), bool) {
		f, ok := path_func[path]
		return f, ok
	})
}

func (t *Web) Shutdown(ctx ...context.Context) {
	ctx = append(ctx, context.Background())
	_ = t.Server.Shutdown(ctx[0])
}

type WebSync struct {
	Server *http.Server
	// mux    *http.ServeMux
	wrs *WebPath
}

func NewSyncMap(conf *http.Server, m *WebPath, matchFunc ...func(path string) (func(w http.ResponseWriter, r *http.Request), bool)) (o *WebSync) {
	if o, e := NewSyncMapNoPanic(conf, m, matchFunc...); e != nil {
		panic(e)
	} else {
		return o
	}
}

func NewSyncMapNoPanic(conf *http.Server, m *WebPath, matchFunc ...func(path string) (func(w http.ResponseWriter, r *http.Request), bool)) (o *WebSync, err error) {

	o = new(WebSync)

	conf.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, m, c)
	}
	o.Server = conf
	o.wrs = m

	matchFunc = append(matchFunc, o.wrs.Load)

	var ln net.Listener
	if tmp, err := net.Listen("tcp", conf.Addr); err != nil {
		return nil, err
	} else {
		ln = tmp
	}
	if conf.TLSConfig != nil {
		ln = tls.NewListener(ln, conf.TLSConfig)
	}

	go func() {
		_ = o.Server.Serve(ln)
	}()

	if o.Server.Handler == nil {
		// o.mux = http.NewServeMux()
		// o.Server.Handler = o.mux

		o.Server.Handler = NewHandler(matchFunc[0])
	}

	// o.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	f, ok := matchFunc[0](r.URL.Path)
	// 	if ok {
	// 		f(w, r)
	// 	} else {
	// 		WithStatusCode(w, http.StatusNotFound)
	// 	}
	// })

	return
}

func (t *WebSync) Shutdown(ctx ...context.Context) {
	ctx = append(ctx, context.Background())
	_ = t.Server.Shutdown(ctx[0])
}

type Handler struct {
	DealF func(path string) (func(w http.ResponseWriter, r *http.Request), bool)
}

func NewHandler(dealF func(path string) (func(w http.ResponseWriter, r *http.Request), bool)) *Handler {
	return &Handler{
		DealF: dealF,
	}
}

func (t *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f, ok := t.DealF(r.URL.Path); ok {
		f(w, r)
	} else {
		WithStatusCode(w, http.StatusNotFound)
	}
}

type WebPath struct {
	Path string `json:"path"`
	// current net.Conn: conn, ok := r.Context().Value(&WebPath).(net.Conn)
	f       func(w http.ResponseWriter, r *http.Request)
	PerSame *WebPath `json:"-"`
	Per     *WebPath `json:"-"`
	Same    *WebPath `json:"same"`
	Next    *WebPath `json:"next"`
	l       sync.RWMutex
}

// WebSync
func (t *WebPath) Reset() {
	t.l.Lock()
	defer t.l.Unlock()
	t.Path = ""
	t.Per = nil
	t.Same = nil
	t.Next = nil
}

// WebSync
func (t *WebPath) GetConn(r *http.Request) net.Conn {
	return r.Context().Value(t).(net.Conn)
}

// O /../../1 => /../../1
//
// X /../../1 => /../../
//
// X /../../1 => /../
func (t *WebPath) Load(path string) (func(w http.ResponseWriter, r *http.Request), bool) {
	if len(path) == 0 || path[0] != '/' {
		return nil, false
	}

	t.l.RLock()
	defer t.l.RUnlock()

	if key, left, fin := parsePath(path); t.Path == key {
		if fin {
			return t.f, t.f != nil
		} else {
			if t.Same != nil {
				return t.Same.Load(left)
			} else {
				return nil, false
			}
		}
	} else {
		if t.Next != nil {
			return t.Next.Load(path)
		} else {
			return nil, false
		}
	}
}

// O /../../1 => /../../1
//
// O /../../1 => /../../
//
// X /../../1 => /../
func (t *WebPath) LoadOnePerfix(path string) (f func(w http.ResponseWriter, r *http.Request), ok bool) {
	if len(path) == 0 || path[0] != '/' {
		return nil, false
	}

	t.l.RLock()
	defer t.l.RUnlock()

	if key, left, fin := parsePath(path); t.Path == key {
		if t.Path == "/" || fin {
			f = t.f
		}
		if t.Same != nil {
			if f1, ok := t.Same.LoadOnePerfix(left); ok {
				f = f1
			}
			return f, f != nil
		} else {
			return f, f != nil
		}
	} else {
		if t.Path == "/" && fin {
			f = t.f
		}
		if t.Next != nil {
			if f1, ok := t.Next.LoadOnePerfix(path); ok {
				f = f1
			}
			return f, f != nil
		} else {
			return f, f != nil
		}
	}
}

// O /../../1 => /../../1
//
// O /../../1 => /../../
//
// O /../../1 => /../
func (t *WebPath) LoadPerfix(path string) (f func(w http.ResponseWriter, r *http.Request), ok bool) {
	if len(path) == 0 || path[0] != '/' {
		return nil, false
	}

	t.l.RLock()
	defer t.l.RUnlock()

	if key, left, fin := parsePath(path); t.Path == "/" {
		f = t.f
		if t.Path != key {
			if t.Next != nil {
				if f1, ok := t.Next.LoadPerfix(path); ok {
					f = f1
				}
			}
		}
		return f, f != nil
	} else if t.Path == key {
		if fin {
			f = t.f
			return f, f != nil
		}
		if t.Same != nil {
			if f1, ok := t.Same.LoadPerfix(left); ok {
				f = f1
				return f, f != nil
			}
		}
		if t.Next != nil {
			if f1, ok := t.Next.LoadPerfix(path); ok {
				f = f1
			}
		}
		return f, f != nil
	} else {
		if t.Next != nil {
			if f1, ok := t.Next.LoadPerfix(path); ok {
				f = f1
			}
		}
		return f, f != nil
	}
}

func parsePath(path string) (key string, left string, fin bool) {
	if pi := 1 + strings.Index(path[1:], "/"); pi != 0 {
		return path[:pi], path[pi:], false
	} else {
		return path, "", true
	}
}

func (t *WebPath) Store(path string, f func(w http.ResponseWriter, r *http.Request)) {
	if len(path) == 0 || path[0] != '/' {
		return
	}

	if f == nil {
		t.Delete(path)
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	if key, left, fin := parsePath(path); t.Path == "" {
		t.Path = key
		// self
		if fin {
			t.f = f
			return
		} else {
			t.Same = &WebPath{PerSame: t}
			t.Same.Store(left, f)
			return
		}
	} else if t.Path == key {
		// same or self
		if fin {
			// self
			t.f = f
			return
		} else {
			// same
			if t.Same != nil {
				t.Same.Store(left, f)
				return
			} else {
				t.Same = &WebPath{PerSame: t}
				t.Same.Store(left, f)
				return
			}
		}
	} else {
		// next
		if t.Next != nil {
			t.Next.Store(path, f)
			return
		} else {
			t.Next = &WebPath{
				Per: t,
			}
			t.Next.Store(path, f)
		}
	}
}

func (t *WebPath) Delete(path string) (deleteMe bool) {
	if len(path) == 0 || path[0] != '/' {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	if key, left, fin := parsePath(path); t.Path == key {
		if fin {
			t.f = nil
			return t.f == nil && t.Next == nil && t.Same == nil
		} else {
			if t.Same != nil {
				if t.Same.Delete(left) {
					t.Same = nil
				}
				return t.f == nil && t.Next == nil && t.Same == nil
			} else {
				return false
			}
		}
	} else {
		if t.Next != nil {
			if t.Next.Delete(path) {
				t.Next = nil
			}
			return t.f == nil && t.Next == nil && t.Same == nil
		} else {
			return false
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

// Deprecated: 反直觉的方法
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
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
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
	g   psync.MapExceeded[string, *[]byte]
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
		i, e = t.raw.Write(b)
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

type WithCacheWiter struct {
	cw  *pio.CacheWriter
	raw http.ResponseWriter
}

func (t *WithCacheWiter) Header() http.Header {
	if t.raw != nil {
		return t.raw.Header()
	}
	return make(http.Header)
}
func (t *WithCacheWiter) Write(b []byte) (i int, e error) {
	if t.cw != nil {
		return t.cw.Write(b)
	}
	return t.raw.Write(b)
}
func (t *WithCacheWiter) WriteHeader(i int) {
	if t.raw != nil {
		t.raw.WriteHeader(i)
	}
}

type Exprier struct {
	max int
	m   psync.Map
	mc  chan string
}

var (
	ErrExpried = errors.New("ErrExpried")
	ErrNoFound = errors.New("ErrNoFound")
)

func NewExprier(max int) *Exprier {
	return &Exprier{
		max: max,
		mc:  make(chan string, max),
	}
}

func (t *Exprier) SetMax(max int) {
	t.max = max
	t.mc = make(chan string, max)
}

func (t *Exprier) Reg(dur time.Duration, reNewKey ...string) (string, error) {
	if t.max <= 0 {
		return "noExprie", nil
	}
	if len(reNewKey) != 0 && reNewKey[0] != "" {
		if _, ok := t.m.Load(reNewKey[0]); ok {
			t.m.Store(reNewKey[0], time.Now().Add(dur))
			return reNewKey[0], nil
		} else {
			return reNewKey[0], ErrNoFound
		}
	} else {
		newkey := uuid.NewString()
		select {
		case t.mc <- newkey:
			t.m.Store(newkey, time.Now().Add(dur))
			return newkey, nil
		default:
			for {
				select {
				case key1 := <-t.mc:
					if t.m.Delete(key1) {
						t.mc <- newkey
						t.m.Store(newkey, time.Now().Add(dur))
						return newkey, nil
					}
				default:
					t.mc <- newkey
					return newkey, nil
				}
			}
		}
	}
}

func (t *Exprier) Check(key string) (time.Time, error) {
	if t.max <= 0 {
		return time.Now(), nil
	}
	if key == "" {
		return time.Now(), ErrNoFound
	}
	ey, ok := t.m.LoadV(key).(time.Time)
	if !ok {
		return time.Now(), ErrNoFound
	} else if time.Now().After(ey) {
		t.m.Delete(key)
		return time.Now(), ErrExpried
	}
	return ey, nil
}

func (t *Exprier) LoopCheck(ctx context.Context, key string, whenfail func(key string, e error)) (e error) {
	if t.max <= 0 {
		return nil
	}
	if _, e := t.Check(key); e != nil {
		whenfail(key, e)
		return e
	}
	go func() {
		for {
			if ey, e := t.Check(key); e != nil {
				whenfail(key, e)
				return
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Until(ey) + time.Second):
				}
			}
		}
	}()
	return nil
}

func (t *Exprier) Disable() {
	t.max = 0
	t.m.ClearAll()
}

func (t *Exprier) Drop(key string) {
	t.m.Delete(key)
}

func (t *Exprier) Len() int {
	return t.m.Len()
}

func WithFlush(w http.ResponseWriter) http.ResponseWriter {
	return withflush{w}
}

func WithCache(w http.ResponseWriter, maxWait uint32) *WithCacheWiter {
	return &WithCacheWiter{raw: w, cw: pio.NewCacheWriter(w, maxWait)}
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

func NotModified(r *http.Request, w http.ResponseWriter, modTime time.Time) (notMod bool) {
	modTimeS := modTime.Format(time.RFC1123)
	modTimeE := modTime.Format(time.RFC3339)

	w.Header().Add(`ETag`, modTimeE)
	w.Header().Add(`Last-Modified`, modTimeS)

	if inm := r.Header.Get(`If-None-Match`); inm == modTimeE {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	if ims := r.Header.Get(`If-Modified-Since`); ims == modTimeS {
		w.WriteHeader(http.StatusNotModified)
		return true
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
			_ = s.Server.Shutdown(context.Background())
		},
	})
	return s
}
