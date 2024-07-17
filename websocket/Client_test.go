package part

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	web "github.com/qydysky/part/web"
)

func Test_Client(t *testing.T) {
	s := New_server()
	{
		ws_mq := s.Interface()

		ws_mq.Pull_tag(map[string]func(interface{}) bool{
			`error`: func(data interface{}) bool {
				return true
			},
			`recv`: func(data interface{}) bool {
				if tmp, ok := data.(Uinterface); ok {
					t.Log(tmp.Id, `=>`, string(tmp.Data))
					t.Log(string(tmp.Data), `=>`, tmp.Id)
					ws_mq.Push_tag(`send`, Uinterface{ //just reply
						Id:   tmp.Id,
						Data: tmp.Data,
					})
				}
				return false
			},
		})
	}

	w := web.New(&http.Server{
		Addr:         "127.0.0.1:10888",
		WriteTimeout: time.Second * time.Duration(10),
	})
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			conn := s.WS(w, r)
			id := <-conn
			t.Log(`user connect!`, id)
			<-conn
			t.Log(`user disconnect!`, id)
		},
	})

	time.Sleep(time.Second)

	{
		c, e := New_client(&Client{
			Url: "ws://127.0.0.1:10888/ws",
			Func_normal_close: func() {
				t.Log("close")
			},
			TO:    1000,
			WTOMs: 1000,
		})
		if e != nil {
			t.Fatal(e)
		}

		ws, e := c.Handle()
		if e != nil {
			t.Fatal(e)
		}

		ws.Pull_tag_only(`rec`, func(wm *WsMsg) (disable bool) {
			if string(wm.Msg) != "test" {
				t.Fatal()
			}
			return false
		})
		ws.PushLock_tag(`send`, &WsMsg{
			Msg: []byte("test"),
		})

		go func() {
			time.Sleep(time.Second * 2)
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

		if !strings.Contains(c.Error().Error(), `i/o`) {
			t.Fatal()
		}

		time.Sleep(time.Second)
	}
	{
		c, e := New_client(&Client{
			Url: "ws://127.0.0.1:10888/ws",
			Func_normal_close: func() {
				t.Log("close")
			},
			TO: 2000,
		})
		if e != nil {
			t.Fatal(e)
		}

		ws, e := c.Handle()
		if e != nil {
			t.Fatal(e)
		}

		ws.Pull_tag_only(`rec`, func(wm *WsMsg) (disable bool) {
			if string(wm.Msg) != "test" {
				t.Fatal()
			}
			return false
		})
		ws.PushLock_tag(`send`, &WsMsg{
			Msg: []byte("test"),
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
