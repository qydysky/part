package part

import (
	"net"
	"net/http"
    "time"
	"github.com/gorilla/websocket"
	idpool "github.com/qydysky/part/idpool"
	mq "github.com/qydysky/part/msgq"
)

type Server struct {
	ws_mq *mq.Msgq
	userpool *idpool.Idpool
}

type Uinterface struct {
	Id uintptr
	Data []byte
}

func New_server() (*Server) {
	return &Server{
		ws_mq: mq.New(200),//收发通道
		userpool: idpool.New(),//浏览器标签页池
	}
}

func (t *Server) WS(w http.ResponseWriter, r *http.Request) (o chan uintptr) {
	upgrader := websocket.Upgrader{}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.ws_mq.Push_tag(`error`,err)
		return
	}

	o = make(chan uintptr,1)

	//从池中获取本会话id
	User := t.userpool.Get()


	//发送
	t.ws_mq.Pull_tag(map[string]func(interface{})(bool){
		`send`:func(data interface{})(bool){
			if u,ok := data.(Uinterface);ok && u.Id == 0 || u.Id == User.Id{
				if err := ws.WriteMessage(websocket.TextMessage,u.Data);err != nil {
					t.ws_mq.Push_tag(`error`,err)
					return true
				}
			}
			return false
		},
		`close`:func(data interface{})(bool){
			if u,ok := data.(Uinterface);ok && u.Id == 0 || u.Id == User.Id{
				return true
			}
			return false
		},
	})

	//接收
	go func(){
		for {
			ws.SetReadDeadline(time.Now().Add(time.Second*time.Duration(300)))
			if _, message, err := ws.ReadMessage();err != nil {
				if websocket.IsCloseError(err,websocket.CloseGoingAway) {
				} else if err,ok := err.(net.Error);ok && err.Timeout() {
					//Timeout , js will reload html
				} else {
					t.ws_mq.Push_tag(`error`,err)
				}
				t.ws_mq.Push_tag(`close`,Uinterface{
					Id:User.Id,
				})
				break
			} else {
				t.ws_mq.Push_tag(`recv`,Uinterface{
					Id:User.Id,
					Data:message,
				})
			}
		}

		//归还
		t.userpool.Put(User)
		//结束
		ws.Close()
		//通知上层结束，上层使用通道传出阻塞
		close(o)
	}()
	//通知上层本此会话的id
	o <- User.Id
	return
}

//how to use
// ws_mq.Pull_tag(map[string]func(interface{})(bool){
// 	`recv`:func(data interface{})(bool){
// 		if tmp,ok := data.(Uinterface);ok {
// 			log.Println(tmp.Id,string(tmp.Data))

// 			if string(tmp.Data) == `close` {
// 				ws_mq.Push_tag(`close`,Uinterface{//close
// 					Id:0,//close all connect
// 				})
// 				//or 
// 				// ws_mq.Push_tag(`close`,Uinterface{//close
// 				// 	Id:tmp.Id,//close this connect
// 				// })
// 				return false
// 			}

// 			ws_mq.Push_tag(`send`,Uinterface{//just reply
// 				Id:tmp.Id,
// 				Data:tmp.Data,
// 			})
// 			//or
// 			ws_mq.Push_tag(`send`,Uinterface{//just reply
// 				Id:0,//send to all
// 				Data:tmp.Data,
// 			})
// 		}
// 		return false
// 	},
// 	`error`:func(data interface{})(bool){
// 		log.Println(data)
// 		return false
// 	},
// })
func (t *Server) Interface() (*mq.Msgq) {
	return t.ws_mq
}

func (t *Server) Len() uint {
	return t.userpool.Len()
}