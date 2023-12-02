package part

import (
	"bytes"
	"encoding/json"
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
