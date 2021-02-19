package part

import (
	"strconv"
	"testing"
	"net/http"
	"time"
	"github.com/skratchdot/open-golang/open"
	web "github.com/qydysky/part/web"
)

func Test_Server(t *testing.T) {
	s := New_server()
	{
		num := 5

		ws_mq := s.Interface()
		ws_mq.Pull_tag(map[string]func(interface{})(bool){
			`error`:func(data interface{})(bool){
				t.Log(data)
				return false
			},
			`recv`:func(data interface{})(bool){
				if tmp,ok := data.(Uinterface);ok {
					t.Log(tmp.Id, `=>`,string(tmp.Data))
					t.Log(string(tmp.Data), `=>`,tmp.Id)
					num -= 1
					if num > 0 {
						ws_mq.Push_tag(`send`,Uinterface{//just reply
							Id:tmp.Id,
							Data:append(tmp.Data,[]byte(` get.server:close after `+strconv.Itoa(num)+` s`)...),
						})
					} else {
						ws_mq.Push_tag(`close`,Uinterface{//close
							Id:tmp.Id,
							Data:[]byte(`closeNormal`),
						})
					}
				}
				return false
			},
		})
	}

	w := web.Easy_boot()
	open.Run("http://"+w.Server.Addr)
	w.Handle(map[string]func(http.ResponseWriter,*http.Request){
		`/ws`:func(w http.ResponseWriter,r *http.Request){
			conn := s.WS(w,r)
			id :=<- conn
			t.Log(`user connect!`,id)
			<- conn
			t.Log(`user disconnect!`,id)
		},
	})
	time.Sleep(time.Second*time.Duration(100))
}