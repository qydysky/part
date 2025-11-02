package part

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	idpool "github.com/qydysky/part/idpool"
	mq "github.com/qydysky/part/msgq"
)

type Server struct {
	ws_mq    *mq.MsgType[Uinterface]
	userpool *idpool.Idpool[struct{}]
}

type Uinterface struct {
	Id   uintptr
	Err  error
	Data []byte
}

func New_server() *Server {
	return &Server{
		ws_mq:    mq.NewType[Uinterface](),                              //收发通道
		userpool: idpool.New(func() *struct{} { return new(struct{}) }), //浏览器标签页池
	}
}

// o <-chan uintptr 返回r创建的id,表示开始接入，用于下述事件中
//
// 0. 发送到t.ws_mq `init`
//
// 1. 尝试开始ws连接，如有错误，发送到t.ws_mq `error`
//
// 2. 监听t.ws_mq `send`，并发送
//
// 2. 监听t.ws_mq `close`，控制连接断开
//
// 2. 将接收到的数据发送到t.ws_mq `recv`
//
// 2. 如有错误，发送到t.ws_mq `error`
//
// 3. 连接结束，发送到t.ws_mq `fin`
//
// o <-chan uintptr 将在断开时再次返回id
func (t *Server) WS(w http.ResponseWriter, r *http.Request) (o <-chan uintptr) {

	//从池中获取本会话id
	User := t.userpool.Get()

	t.ws_mq.Push_tag(`init`, Uinterface{
		Id: User.Id,
	})

	co := make(chan uintptr, 5)

	go func() {
		defer func() {
			// 归还
			t.userpool.Put(User)
			// fin事件
			t.ws_mq.Push_tag(`fin`, Uinterface{
				Id: User.Id,
			})
			// 通知上层结束，上层使用通道传出阻塞
			co <- User.Id
		}()

		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.ws_mq.Push_tag(`error`, Uinterface{
				Id:  User.Id,
				Err: err,
			})
			return
		}

		//发送
		t.ws_mq.Pull_tag(map[string]func(Uinterface) bool{
			`send`: func(u Uinterface) bool {
				if u.Id == 0 || u.Id == User.Id {
					if err := ws.WriteMessage(websocket.TextMessage, u.Data); err != nil {
						t.ws_mq.Push_tag(`error`, Uinterface{
							Id:  User.Id,
							Err: err,
						})
						return true
					}
				}
				return false
			},
			`close`: func(u Uinterface) bool {
				if u.Id == 0 || u.Id == User.Id {
					msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, string(u.Data))
					TO := time.Now().Add(time.Second * time.Duration(5))

					if err := ws.WriteControl(websocket.CloseMessage, msg, TO); err != nil && !errors.Is(err, websocket.ErrCloseSent) {
						t.ws_mq.Push_tag(`error`, Uinterface{
							Id:  User.Id,
							Err: err,
						})
					}
					return true
				}
				return false
			},
		})

		//通知上层本此会话的id
		co <- User.Id

		for {
			_ = ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(300)))
			if _, message, err := ws.ReadMessage(); err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					//client close
				} else if e, ok := err.(net.Error); ok && e.Timeout() {
					//Timeout
				} else {
					//other
					t.ws_mq.Push_tag(`error`, Uinterface{
						Id:  User.Id,
						Err: err,
					})
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
		t.ws_mq.Push_tag(`close`, Uinterface{
			Id: User.Id,
		})

		//结束
		ws.Close()

	}()

	return co
}

// how to use
//
//	{
//		ws_mq := s.Interface()
//		// close all
//		defer ws_mq.Push_tag(`close`, Uinterface{
//			Id: 0,
//		})
//
//		ws_mq.Pull_tag(map[string]func(Uinterface) bool{
//			// 新连接建立
//			`init`: func(u Uinterface) bool {
//				fmt.Println(u.Id, "connected!")
//				return false
//			},
//			`error`: func(u Uinterface) bool {
//				fmt.Println(u.Id, u.Err)
//
//				ws_mq.Push_tag(`send`, Uinterface{
//					Id:   u.Id,
//					Data: []byte("send something"),
//				})
//				ws_mq.Push_tag(`close`, Uinterface{
//					Id: u.Id,
//				})
//				return false
//			},
//			// 从客户端接收数据
//			`recv`: func(u Uinterface) bool {
//				t.Log(u.Id, `=>`, string(u.Data))
//				return false
//			},
//			// 连接断开
//			`fin`: func(u Uinterface) bool {
//				fmt.Println(u.Id, "fin!")
//				return false
//			},
//		})
//	}
func (t *Server) Interface() *mq.MsgType[Uinterface] {
	return t.ws_mq
}

// 当前连接数
func (t *Server) Len() int64 {
	return t.userpool.InUse()
}
