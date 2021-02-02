package part

import (
	"testing"
	"net/http"
	"time"
	"github.com/skratchdot/open-golang/open"
	web "github.com/qydysky/part/web"
)

func Test_Server(t *testing.T) {
	s := New_server()
	{
		ws_mq := s.Interface()
		ws_mq.Pull_tag(map[string]func(interface{})(bool){
			`recv`:func(data interface{})(bool){
				if tmp,ok := data.(Uinterface);ok {
					t.Log(tmp.Id,string(tmp.Data))
					ws_mq.Push_tag(`send`,Uinterface{//just reply
						Id:tmp.Id,
						Data:tmp.Data,
					})
				}
				return false
			},
		})
	}

	w := web.Easy_boot()
	open.Run("http://"+w.Server.Addr)
	w.Handle(map[string]func(http.ResponseWriter,*http.Request){
		`/ws`:func(w http.ResponseWriter,r *http.Request){
			s.WS(w,r)
		},
	})
	time.Sleep(time.Second*time.Duration(100))
}