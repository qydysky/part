package part

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	web "github.com/qydysky/part/web"
)

func Test_Server(t *testing.T) {
	t.Parallel()

	s := New_server()
	{
		ws_mq := s.Interface()
		// close all
		defer ws_mq.Push_tag(`close`, Uinterface{
			Id: 0,
		})
		ws_mq.Pull_tag(map[string]func(Uinterface) bool{
			`init`: func(u Uinterface) bool {
				fmt.Println(u.Id, "connected!")
				return false
			},
			`error`: func(u Uinterface) bool {
				fmt.Println(u.Id, u.Err)
				ws_mq.Push_tag(`close`, Uinterface{
					Id: u.Id,
				})
				return false
			},
			`recv`: func(u Uinterface) bool {
				t.Log(u.Id, `=>`, string(u.Data))
				if u.Data[0] != '1' {
					t.Fatal()
				}
				ws_mq.Push_tag(`send`, Uinterface{
					Id:   u.Id,
					Data: []byte{'2'},
				})
				ws_mq.Push_tag(`close`, Uinterface{
					Id: u.Id,
				})
				return false
			},
			`fin`: func(u Uinterface) bool {
				fmt.Println(u.Id, "fin!")
				return false
			},
		})
	}

	w := web.Easy_boot()
	defer w.Shutdown()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			conn := s.WS(w, r)
			<-conn
			<-conn
		},
	})

	time.Sleep(time.Second)

	if c, e := New_client(&Client{
		Url: `ws://` + w.Server.Addr + `/ws`,
	}); e != nil {
		t.Fatal(e)
	} else if handler, e := c.Handle(); e != nil {
		t.Fatal(e)
	} else {
		handler.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				if b[0] != '2' {
					t.Fatal()
				}
				t.Log("ser", string(b))
				return nil
			})
			return false
		})
		handler.Push_tag(`send`, &WsMsg{
			Msg: func(f func([]byte) error) error {
				return f([]byte{'1'})
			},
		})
		cancle, c := handler.Pull_tag_chan(`exit`, 1, t.Context())
		<-c
		cancle()
	}
}
