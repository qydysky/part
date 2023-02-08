package part

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	s := Easy_boot()
	t.Log(`http://` + s.Server.Addr)
	time.Sleep(time.Second * time.Duration(100))
}

func Test_ServerSyncMap(t *testing.T) {
	var m WebPath
	m.Store("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1"))
	})
	NewSyncMap(&http.Server{
		Addr: "127.0.0.1:9090",
	}, &m)
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		m.Store("/1", func(w http.ResponseWriter, r *http.Request) {

			type d struct {
				A string         `json:"a"`
				B []string       `json:"b"`
				C map[string]int `json:"c"`
			}

			t := strconv.Itoa(i)

			ResStruct{0, "ok", d{t, []string{t}, map[string]int{t: 1}}}.Write(w)
		})
	}
}

func BenchmarkXxx(b *testing.B) {
	var m WebPath
	type d struct {
		A string `json:"path"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store("/", func(w http.ResponseWriter, r *http.Request) {
			ResStruct{0, "ok", d{"/"}}.Write(w)
		})
	}
}

func Test_ServerSyncMapP(t *testing.T) {
	var m WebPath
	type d struct {
		A string `json:"path"`
	}

	NewSyncMap(&http.Server{
		Addr: "127.0.0.1:9090",
	}, &m)
	m.Store("/", func(w http.ResponseWriter, r *http.Request) {
		ResStruct{0, "ok", d{"/"}}.Write(w)
	})
	m.Store("/1/2", func(w http.ResponseWriter, r *http.Request) {
		ResStruct{0, "ok", d{"/1/2"}}.Write(w)
	})
	m.Store("/1", func(w http.ResponseWriter, r *http.Request) {
		ResStruct{0, "ok", d{"/1"}}.Write(w)
	})
	m.Store("/2", func(w http.ResponseWriter, r *http.Request) {
		ResStruct{0, "ok", d{"/2"}}.Write(w)
	})

	time.Sleep(time.Second * time.Duration(23))
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
