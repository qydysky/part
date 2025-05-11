package part

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	idpool "github.com/qydysky/part/idpool"
	mq "github.com/qydysky/part/msgq"
)

type Server struct {
	ws_mq    *mq.Msgq
	userpool *idpool.Idpool[struct{}]
	m        sync.Mutex
}

type Uinterface struct {
	Id   uintptr
	Data []byte
}

type uinterface struct { //内部消息
	Id   uintptr
	Data interface{}
}

func New_server() *Server {
	return &Server{
		ws_mq:    mq.New(),                                              //收发通道
		userpool: idpool.New(func() *struct{} { return new(struct{}) }), //浏览器标签页池
	}
}

func (t *Server) WS(w http.ResponseWriter, r *http.Request) (o chan uintptr) {
	upgrader := websocket.Upgrader{}

	o = make(chan uintptr, 1)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		close(o)
		t.ws_mq.Push_tag(`error`, err)
		return
	}

	//从池中获取本会话id
	User := t.userpool.Get()

	//发送
	t.ws_mq.Pull_tag(map[string]func(interface{}) bool{
		`send`: func(data interface{}) bool {
			if u, ok := data.(Uinterface); ok && u.Id == 0 || u.Id == User.Id {
				t.m.Lock()
				defer t.m.Unlock()
				if err := ws.WriteMessage(websocket.TextMessage, u.Data); err != nil {
					t.ws_mq.Push_tag(`error`, err)
					return true
				}
			}
			return false
		},
		`close`: func(data interface{}) bool {
			if u, ok := data.(Uinterface); ok && u.Id == 0 || u.Id == User.Id { //服务器主动关闭
				msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, string(u.Data))
				TO := time.Now().Add(time.Second * time.Duration(5))

				if err := ws.WriteControl(websocket.CloseMessage, msg, TO); err != nil {
					t.ws_mq.Push_tag(`error`, err)
				}
				return true
			} else if u, ok := data.(uinterface); ok { //接收发生错误关闭
				return ok && u.Data.(string) == `rev_close` && u.Id == 0 || u.Id == User.Id
			}
			return false
		},
	})

	//接收
	go func() {
		for {
			ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(300)))
			if _, message, err := ws.ReadMessage(); err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					//client close
				} else if e, ok := err.(net.Error); ok && e.Timeout() {
					//Timeout
				} else {
					//other
					t.ws_mq.Push_tag(`error`, err)
				}
				break
			} else {
				t.ws_mq.Push_tag(`recv`, Uinterface{
					Id:   User.Id,
					Data: message,
				})
			}
		}

		//接收发生错误，通知发送关闭
		t.ws_mq.Push_tag(`close`, uinterface{
			Id:   User.Id,
			Data: `rev_close`,
		})
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

// how to use
//
//	ws_mq.Pull_tag(map[string]func(interface{})(bool){
//		`recv`:func(data interface{})(bool){
//			if tmp,ok := data.(Uinterface);ok {
//				log.Println(tmp.Id,string(tmp.Data))
//
//				if string(tmp.Data) == `close` {
//					ws_mq.Push_tag(`close`,Uinterface{//close
//						Id:0,//close all connect
//					})
//					//or
//					// ws_mq.Push_tag(`close`,Uinterface{//close
//					// 	Id:tmp.Id,//close this connect
//					// })
//					return false
//				}
//
//					ws_mq.Push_tag(`send`,Uinterface{//just reply
//						Id:tmp.Id,
//						Data:tmp.Data,
//					})
//					//or
//					ws_mq.Push_tag(`send`,Uinterface{//just reply
//						Id:0,//send to all
//						Data:tmp.Data,
//					})
//				}
//				return false
//			},
//			`error`:func(data interface{})(bool){
//				log.Println(data)
//				return false
//			},
//		})
func (t *Server) Interface() *mq.Msgq {
	return t.ws_mq
}

func (t *Server) Len() int64 {
	return t.userpool.InUse()
}
