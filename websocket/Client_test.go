package part

import (
	"context"
	"net/http"
	"testing"
	"time"

	web "github.com/qydysky/part/web"
)

func Test_Client(t *testing.T) {
	s := New_server()
	{
		ws_mq := s.Interface()

		ws_mq.Pull_tag(map[string]func(Uinterface) bool{
			`error`: func(data Uinterface) bool {
				return true
			},
			`recv`: func(tmp Uinterface) bool {
				t.Log(tmp.Id, `=>`, string(tmp.Data))
				t.Log(string(tmp.Data), `=>`, tmp.Id)
				ws_mq.Push_tag(`send`, Uinterface{ //just reply
					Id:   tmp.Id,
					Data: tmp.Data,
				})
				return false
			},
		})
	}

	w := web.New(&http.Server{
		Addr:         "127.0.0.1:10888",
		WriteTimeout: time.Second * time.Duration(10),
	})
	defer w.Shutdown()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET")
				return
			}
			conn := s.WS(w, r)
			id := <-conn
			t.Log(`user connect!`, id)
			<-conn
			t.Log(`user disconnect!`, id)
		},
	})

	// wait
	time.Sleep(time.Second)

	{
		c, e := New_client(&Client{
			Url: "ws://127.0.0.1:10888/ws",
			Func_normal_close: func() {
				t.Log("close")
			},
			WTOMs: 1000,
		})
		if e != nil {
			t.Fatal(e)
		}

		ws, e := c.Handle()
		if e != nil {
			t.Fatal(e)
		}

		ws.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				if string(b) != "test" {
					t.Fatal()
				}
				return nil
			})
			return false
		})
		ws.PushLock_tag(`send`, &WsMsg{
			Msg: func(f func([]byte) error) error {
				return f([]byte("test"))
			},
		})

		time.AfterFunc(time.Second*2, c.Close)

		{
			cancel, c := ws.Pull_tag_chan(`exit`, 1, context.Background())
			<-c
			cancel()
			t.Log("exit")
		}
	}
	{
		c, e := New_client(&Client{
			Url: "ws://127.0.0.1:10888/ws",
			Func_normal_close: func() {
				t.Log("close")
			},
			RTOMs: 2000,
		})
		if e != nil {
			t.Fatal(e)
		}

		ws, e := c.Handle()
		if e != nil {
			t.Fatal(e)
		}

		ws.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				if string(b) != "test" {
					t.Fatal()
				}
				return nil
			})
			return false
		})
		ws.PushLock_tag(`send`, &WsMsg{
			Msg: func(f func([]byte) error) error {
				return f([]byte("test"))
			},
		})

		go func() {
			time.Sleep(time.Second)
			t.Log("call close")
			c.Close()
			t.Log("call close done")
		}()

		{
			cancel, c := ws.Pull_tag_chan(`exit`, 1, context.Background())
			<-c
			cancel()
			t.Log("exit")
		}

		if c.Error() != nil {
			t.Fatal(c.Error())
		}

		time.Sleep(time.Second)
	}
}

func Test_Client2(t *testing.T) {
	s := New_server()
	{
		ws_mq := s.Interface()

		ws_mq.Pull_tag(map[string]func(Uinterface) bool{
			`error`: func(data Uinterface) bool {
				return true
			},
			`recv`: func(tmp Uinterface) bool {
				t.Log(tmp.Id, `=>`, string(tmp.Data))
				t.Log(string(tmp.Data), `=>`, tmp.Id)
				return false
			},
		})
	}

	w := web.New(&http.Server{
		Addr:         "127.0.0.1:10888",
		WriteTimeout: time.Second * time.Duration(10),
	})
	defer w.Shutdown()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET")
				return
			}
			conn := s.WS(w, r)
			id := <-conn
			t.Log(`user connect!`, id)
			<-conn
			t.Log(`user disconnect!`, id)
		},
	})

	// wait
	time.Sleep(time.Second)

	{
		c, e := New_client(&Client{
			Url: "ws://127.0.0.1:10888/ws",
			Func_normal_close: func() {
				t.Log("close")
			},
			WTOMs: 7000,
		})
		if e != nil {
			t.Fatal(e)
		}

		ws, e := c.Handle()
		if e != nil {
			t.Fatal(e)
		}

		go ws.Push_tag(`send`, &WsMsg{
			Msg: func(f func([]byte) error) error {
				return f([]byte("test"))
			},
		})
		go c.Close()

		{
			cancel, c := ws.Pull_tag_chan(`exit`, 1, context.Background())
			<-c
			cancel()
			t.Log("exit")
		}
	}
}
