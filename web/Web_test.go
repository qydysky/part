package part

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	pio "github.com/qydysky/part/io"
	reqf "github.com/qydysky/part/reqf"
)

func Test_Exprier(t *testing.T) {
	exp := NewExprier(1)
	if key, e := exp.Reg(time.Second); e != nil {
		t.Fail()
	} else {
		if _, e := exp.Check(key); e != nil {
			t.Fail()
		}
		if k, e := exp.Reg(time.Second); e != nil {
			t.Fail()
		} else {
			if _, e := exp.Check(key); !errors.Is(e, ErrNoFound) {
				t.Fail()
			}
			key = k
		}
		time.Sleep(time.Second * 2)
		if _, e := exp.Check(key); !errors.Is(e, ErrExpried) {
			t.Fail()
		}
	}
}

func Test_Server(t *testing.T) {
	s := New(&http.Server{
		Addr:         "127.0.0.1:13000",
		WriteTimeout: time.Second * time.Duration(10),
	})
	defer s.Shutdown()
	s.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/no`: func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("abc强强强强"))
		},
	})

	time.Sleep(time.Second)

	r := reqf.New()
	{
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13000/no",
		})
		if !bytes.Equal(r.Respon, []byte("abc强强强强")) {
			t.Fatal(r.Respon)
		}
	}
}

func failIfNot[T comparable](t *testing.T, a, b T) {
	t.Logf("a:'%v' b:'%v'", a, b)
	if a != b {
		t.Fail()
	}
}

func Test_path(t *testing.T) {
	var m WebPath
	var res string
	var f1 = func(_ http.ResponseWriter, _ *http.Request) { res += "f1" }
	var f2 = func(_ http.ResponseWriter, _ *http.Request) { res += "f2" }
	m.Store("/1", f2)
	m.Store("/1/", f1)
	failIfNot(t, res, "")
	if sf1, ok := m.LoadPerfix("/1/"); ok {
		sf1(nil, nil)
	}
	failIfNot(t, res, "f1")
	if sf1, ok := m.LoadPerfix("/1"); ok {
		sf1(nil, nil)
	}
	failIfNot(t, res, "f1f2")
	if sf1, ok := m.LoadPerfix("/121"); ok {
		sf1(nil, nil)
	}
	failIfNot(t, res, "f1f2")
	if sf1, ok := m.LoadPerfix("/1/1"); ok {
		sf1(nil, nil)
	}
	failIfNot(t, res, "f1f2f1")
}

func Test_Store(t *testing.T) {
	var webPath = WebPath{}
	var res = ""
	var f = func(i string) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			res += i
		}
	}
	var checkFull = func(webPath *WebPath) {
		res = ""
		if _, ok := webPath.Load("/1/2"); ok {
			t.Fatal()
		}

		if _, ok := webPath.Load("/1/"); ok {
			t.Fatal()
		}

		if _, ok := webPath.Load("/2"); ok {
			t.Fatal()
		}

		if f, ok := webPath.Load("/1/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.Load("/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.Load("/"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if res != "cba" {
			t.Fatal()
		}
	}
	var checkOnePerfix = func(webPath *WebPath) {
		res = ""
		if _, ok := webPath.LoadOnePerfix("/1/2"); ok {
			t.Fatal()
		}

		if _, ok := webPath.LoadOnePerfix("/1/"); ok {
			t.Fatal()
		}

		if f, ok := webPath.LoadOnePerfix("/2"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadOnePerfix("/1/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadOnePerfix("/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadOnePerfix("/"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if res != "acba" {
			t.Fatal()
		}
	}
	var checkPerfix = func(webPath *WebPath) {
		res = ""
		if f, ok := webPath.LoadPerfix("/1/2"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadPerfix("/1/"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadPerfix("/2"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadPerfix("/1/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadPerfix("/1"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if f, ok := webPath.LoadPerfix("/"); !ok {
			t.Fatal()
		} else {
			f(nil, nil)
		}

		if res != "aaacba" {
			t.Fatal(res)
		}
	}

	webPath.Store("/", f("a"))
	webPath.Store("/1", f("b"))
	webPath.Store("/1/", f("b"))
	webPath.Store("/1/1", f("b"))
	if m, e := json.Marshal(webPath); e != nil {
		t.Fatal(e)
	} else if string(m) != `{"path":"/","same":null,"next":{"path":"/1","same":{"path":"/","same":null,"next":{"path":"/1","same":null,"next":null}},"next":null}}` {
		t.Fatal(string(m))
	}
	webPath.Reset()
	t.Log(0)

	webPath.Store("/", f("a"))
	webPath.Store("/1", f("b"))
	webPath.Store("/2", f("b"))
	webPath.Store("/1/1", f("c"))
	webPath.Store("/1/2", f("d"))
	webPath.Delete("/2")
	webPath.Delete("/1/2")
	if m, e := json.Marshal(webPath); e != nil {
		t.Fatal(e)
	} else if string(m) != `{"path":"/","same":null,"next":{"path":"/1","same":{"path":"/1","same":null,"next":null},"next":null}}` {
		t.Fatal(string(m))
	}

	checkFull(&webPath)
	checkOnePerfix(&webPath)
	checkPerfix(&webPath)
	webPath.Reset()
	t.Log(1)

	webPath.Store("/1", f("b"))
	webPath.Store("/", f("a"))
	webPath.Store("/1/1", f("c"))
	if m, e := json.Marshal(webPath); e != nil {
		t.Fatal(e)
	} else if string(m) != `{"path":"/1","same":{"path":"/1","same":null,"next":null},"next":{"path":"/","same":null,"next":null}}` {
		t.Fatal(string(m))
	}

	checkFull(&webPath)
	checkOnePerfix(&webPath)
	checkPerfix(&webPath)
	webPath.Reset()
	t.Log(2)

	webPath.Store("/1/1", f("c"))
	webPath.Store("/", f("a"))
	webPath.Store("/1", f("b"))
	if m, e := json.Marshal(webPath); e != nil {
		t.Fatal(e)
	} else if string(m) != `{"path":"/1","same":{"path":"/1","same":null,"next":null},"next":{"path":"/","same":null,"next":null}}` {
		t.Fatal(string(m))
	}

	checkFull(&webPath)
	checkOnePerfix(&webPath)
	checkPerfix(&webPath)
	webPath.Reset()
}

func Test_Server2(t *testing.T) {
	var m WebPath
	m.Store("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("/"))
	})
	m.Store("/1", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("/1"))
	})
	s := NewSyncMap(&http.Server{
		Addr:         "127.0.0.1:13001",
		WriteTimeout: time.Millisecond,
	}, &m, m.LoadPerfix)
	defer s.Shutdown()

	time.Sleep(time.Second)

	r := reqf.New()
	{
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13001/1",
		})
		if !bytes.Equal(r.Respon, []byte("/1")) {
			t.Fatal(r.Respon)
		}
	}
	{
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13001/2",
		})
		if !bytes.Equal(r.Respon, []byte("/")) {
			t.Fatal(r.Respon)
		}
	}
}

func Test_ServerSyncMap(t *testing.T) {
	var m WebPath
	m.Store("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("1"))
	})
	s := NewSyncMap(&http.Server{
		Addr: "127.0.0.1:13000",
	}, &m)
	defer s.Shutdown()
	m.Store("/1", func(w http.ResponseWriter, _ *http.Request) {

		type d struct {
			A string         `json:"a"`
			B []string       `json:"b"`
			C map[string]int `json:"c"`
		}

		ResStruct{0, "ok", d{"0", []string{"0"}, map[string]int{"0": 1}}}.Write(w)
	})
	m.Store("/2/", func(w http.ResponseWriter, _ *http.Request) {
		panic(1)
	})

	time.Sleep(time.Second)

	r := reqf.New()
	{
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13000/1",
		})
		if !bytes.Equal(r.Respon, []byte("{\"code\":0,\"message\":\"ok\",\"data\":{\"a\":\"0\",\"b\":[\"0\"],\"c\":{\"0\":1}}}")) {
			t.Error(string(r.Respon))
		}
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13000/2",
		})
		if r.Response.StatusCode != 404 {
			t.Error(string(r.Respon))
		}
		m.Store("/2/", nil)
		r.Reqf(reqf.Rval{
			Url: "http://127.0.0.1:13000/2/",
		})
		if r.Response.StatusCode != 404 {
			t.Error(string(r.Respon))
		}
	}
}

func Test_ClientBlock(t *testing.T) {
	var m WebPath
	m.Store("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("1"))
	})
	s := NewSyncMap(&http.Server{
		Addr:         "127.0.0.1:13001",
		WriteTimeout: time.Millisecond,
	}, &m)
	defer s.Shutdown()

	m.Store("/to", func(w http.ResponseWriter, r *http.Request) {
		rwc := pio.WithCtxTO(r.Context(), fmt.Sprintf("server handle %v by %v ", r.URL.Path, r.RemoteAddr), time.Second,
			w, r.Body, func(s string) {
				fmt.Println(s)
				if !strings.Contains(s, "write blocking after rw 2s > 1s, goruntime leak") {
					t.Fatal(s)
				}
			})
		defer rwc.Close()

		type d struct {
			A string         `json:"a"`
			B []string       `json:"b"`
			C map[string]int `json:"c"`
		}

		var t = ResStruct{0, "ok", d{"0", []string{"0"}, map[string]int{"0": 1}}}
		data, e := json.Marshal(t)
		if e != nil {
			t.Code = -1
			t.Data = nil
			t.Message = e.Error()
			data, _ = json.Marshal(t)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	time.Sleep(time.Second)

	r := reqf.New()
	{
		rc, wc := io.Pipe()
		c := make(chan struct{})
		go func() {
			time.Sleep(time.Second * 3)
			d, _ := io.ReadAll(rc)
			fmt.Println(string(d))
			fmt.Println(r.Response.Status)
			close(c)
		}()
		r.Reqf(reqf.Rval{
			Url:         "http://127.0.0.1:13001/to",
			SaveToPipe:  &pio.IOpipe{R: rc, W: wc},
			WriteLoopTO: 5000,
			Async:       true,
		})
		<-c
	}
}

func BenchmarkXxx(b *testing.B) {
	var m WebPath
	type d struct {
		A string `json:"path"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store("/", func(w http.ResponseWriter, _ *http.Request) {
			ResStruct{0, "ok", d{"/"}}.Write(w)
		})
	}
}

func Test_ServerSyncMapP(t *testing.T) {
	var m WebPath
	type d struct {
		A string `json:"path"`
	}

	o := NewSyncMap(&http.Server{
		Addr: "127.0.0.1:13002",
	}, &m)
	defer o.Shutdown()
	m.Store("/1/2", func(w http.ResponseWriter, _ *http.Request) {
		ResStruct{0, "ok", d{"/1/2"}}.Write(w)
	})
	m.Store("/1/", func(w http.ResponseWriter, _ *http.Request) {
		ResStruct{0, "ok", d{"/1/"}}.Write(w)
	})
	m.Store("/2/", func(w http.ResponseWriter, _ *http.Request) {
		ResStruct{0, "ok", d{"/2/"}}.Write(w)
	})
	m.Store("/", func(w http.ResponseWriter, _ *http.Request) {
		ResStruct{0, "ok", d{"/"}}.Write(w)
	})
	m.Store("/conn", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if conn, ok := ctx.Value(&m).(net.Conn); ok {
			ResStruct{0, "ok", d{A: conn.RemoteAddr().String()}}.Write(w)
		} else {
			ResStruct{0, "fail", d{"/"}}.Write(w)
		}
	})
	time.Sleep(time.Second)

	r := reqf.New()
	res := ResStruct{}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/conn",
	})
	json.Unmarshal(r.Respon, &res)
	if res.Message != "ok" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/1",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/1/",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/1/" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/2",
	})
	if r.Response.StatusCode != 404 {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/1/23",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/1/" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/1/2/3",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/1/" {
		t.Fatal("")
	}
	r.Reqf(reqf.Rval{
		Url: "http://127.0.0.1:13002/1/2",
	})
	json.Unmarshal(r.Respon, &res)
	if data, ok := res.Data.(map[string]any); !ok || data["path"].(string) != "/1/2" {
		t.Fatal("")
	}
}

type ResStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (t ResStruct) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	data, e := json.Marshal(t)
	if e != nil {
		t.Code = -1
		t.Data = nil
		t.Message = e.Error()
		data, _ = json.Marshal(t)
	}
	w.Write(data)
}

func Test1(b *testing.T) {
	exp := NewExprier(20)

	el := make(chan error, 100)
	for i := 0; i < 20; i++ {
		key, _ := exp.Reg(time.Second)
		e := exp.LoopCheck(context.Background(), key, func(key string, e error) {
			if e != nil {
				el <- e
			}
		})
		if e != nil {
			b.Fatal(e)
		}
	}

	time.Sleep(time.Second * 3)

	for len(el) > 0 {
		if !errors.Is(<-el, ErrExpried) {
			b.Fatal()
		}
	}
}
