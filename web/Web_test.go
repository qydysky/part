package part

import (
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	s := Easy_boot()
	t.Log(`http://` + s.Server.Addr)
	time.Sleep(time.Second * time.Duration(100))
}

func Test_ServerSync(t *testing.T) {
	s := NewSync(&http.Server{
		Addr: "127.0.0.1:9090",
	})
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		s.HandleSync("/1", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(strconv.Itoa(i)))
		})
	}
}

func Test_ServerSyncMap(t *testing.T) {
	var m sync.Map
	m.Store("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1"))
	})
	NewSyncMap(&http.Server{
		Addr: "127.0.0.1:9090",
	}, &m)
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		m.Store("/1", func(w http.ResponseWriter, r *http.Request) {
			t.Log(r.URL.Path)
			t.Log(r.URL.EscapedPath())
			w.Write([]byte(strconv.Itoa(i)))
		})
	}
}
