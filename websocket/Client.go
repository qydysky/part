package part

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/websocket"

	pio "github.com/qydysky/part/io"
	msgq "github.com/qydysky/part/msgq"
	pool "github.com/qydysky/part/pool"
	psync "github.com/qydysky/part/sync"
	us "github.com/qydysky/part/unsafe"
)

type Client struct {
	Url string
	// rec send close
	msg *msgq.MsgType[*WsMsg]

	RTOMs int // default: 300s
	WTOMs int // default: 300s
	// BufSize int               // msg buf 1: always use single buf >1: use bufs cycle. default:10
	Header map[string]string // default: map[string]string{}
	Proxy  string

	Ping  Ping // default: no ping
	pingT int64

	Msg_normal_close  string // default: ``
	Func_normal_close func() // default: func(){}
	Func_abort_close  func() // default: func(){}

	close bool
	err   error

	l sync.RWMutex
}

type WsMsg struct {
	ty  int
	Msg func(func([]byte) error) error
}

type Ping struct {
	Msg      []byte
	Period   int
	had_pong bool
}

func New_client(config *Client) (*Client, error) {
	tmp := Client{
		RTOMs:             300 * 1000,
		WTOMs:             300 * 1000,
		Func_normal_close: func() {},
		Func_abort_close:  func() {},
		// BufSize:           10,
		msg: msgq.NewType[*WsMsg](),
	}
	tmp.Url = config.Url
	if tmp.Url == "" {
		return nil, errors.New(`url == ""`)
	}
	// if v := config.BufSize; v >= 1 {
	// 	tmp.BufSize = v
	// }
	if v := config.RTOMs; v != 0 {
		tmp.RTOMs = v
	}
	if v := config.WTOMs; v != 0 {
		tmp.WTOMs = v
	}
	tmp.Msg_normal_close = config.Msg_normal_close
	tmp.Header = config.Header
	if v := config.Func_normal_close; v != nil {
		tmp.Func_normal_close = v
	}
	if v := config.Func_abort_close; v != nil {
		tmp.Func_abort_close = v
	}
	if v := config.Proxy; v != "" {
		tmp.Proxy = v
	}
	if config.Ping.Period != 0 {
		tmp.Ping = config.Ping
	}
	return &tmp, nil
}

// 处理
//
// // 发送数据到服务器
//
//	msg.Push_tag(`send`, &WsMsg{
//		Msg:  []byte("message"),
//	})
//
// // 接收服务器的数据
//
//	msg.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
//		wm.Msg(func(b []byte) error {
//			fmt.Println(string(b))
//			return nil
//		})
//		return false
//	})
//
// // 主动断开连接
//
// (*Client).Close()
//
// or
//
// msg.Push_tag(`close`, nil)
//
// // 监听退出
//
//	msg.Pull_tag_only(`exit`, func(_ any) (disable bool) {
//		return false
//	})
func (o *Client) Handle() (*msgq.MsgType[*WsMsg], error) {
	tmp_Header := make(http.Header)
	for k, v := range o.Header {
		tmp_Header.Set(k, v)
	}

	dial := websocket.DefaultDialer
	if o.Proxy != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(o.Proxy)
		}
		dial.Proxy = proxy
	}
	c, response, err := dial.Dial(o.Url, tmp_Header)
	if err != nil {
		o.error(err)
	} else if err := c.SetWriteDeadline(time.Now().Add(time.Duration(o.WTOMs * int(time.Millisecond)))); err != nil {
		o.error(err)
	}
	if err != nil {
		e := err.Error()
		if response != nil {
			if response.Status != "" {
				e += ` ` + response.Status
			}
			if response.Body != nil {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					return nil, err
				}
				response.Body.Close()
				e += ` ` + us.B2S(body)
			}
		}
		return nil, errors.New(e)
	}

	// recv
	go func() {
		defer func() {
			o.msg.Push_tag(`exit`, nil)
			o.l.Lock()
			o.close = true
			o.l.Unlock()
		}()

		buf := make([]byte, humanize.KByte)
		var msgsPool = pool.NewPoolBlocks[byte]()
		var err error
		for err == nil {
			if e := c.SetReadDeadline(time.Now().Add(time.Duration(o.RTOMs * int(time.Millisecond)))); e != nil {
				err = e
			} else if msg_type, r, e := c.NextReader(); e != nil {
				err = e
			} else if msg, e := pio.ReadAll(r, buf); e != nil {
				err = e
			} else {
				tmpbuf := msgsPool.Get()
				{
					*tmpbuf = append((*tmpbuf)[:0], msg...)
					switch msg_type {
					case websocket.PingMessage:
						o.msg.Push_tag(`send`, &WsMsg{
							ty: websocket.PongMessage,
							Msg: func(f func([]byte) error) error {
								defer msgsPool.Put(tmpbuf)
								return f(*tmpbuf)
							},
						})
					case websocket.PongMessage:
						msgsPool.Put(tmpbuf)
						o.pingT = time.Now().UnixMilli()
						time.AfterFunc(time.Duration(o.Ping.Period*int(time.Millisecond)), func() {
							o.msg.Push_tag(`send`, &WsMsg{
								ty: websocket.PingMessage,
								Msg: func(f func([]byte) error) error {
									return f(o.Ping.Msg)
								},
							})
						})
						o.Ping.had_pong = true
					default:
						o.msg.Push_tag(`recv`, &WsMsg{
							ty: websocket.TextMessage,
							Msg: func(f func([]byte) error) error {
								defer msgsPool.Put(tmpbuf)
								return f(*tmpbuf)
							},
						})
					}
				}
			}
			if e, ok := err.(*websocket.CloseError); ok {
				switch e.Code {
				case websocket.CloseNormalClosure:
					o.Func_normal_close()
				case websocket.CloseAbnormalClosure:
					o.Func_abort_close()
					o.error(err)
				default:
					o.error(err)
				}
			} else if err != nil {
				o.error(err)
			}
		}
	}()

	// websocket.Conn write not goroutine safe
	var wlock psync.RWMutex

	// send
	o.msg.Pull_tag_only(`send`, func(wm *WsMsg) (disable bool) {
		defer wlock.Lock()()
		if wm.ty == 0 {
			wm.ty = websocket.TextMessage
		}
		if err := c.SetWriteDeadline(time.Now().Add(time.Duration(o.WTOMs * int(time.Millisecond)))); err != nil {
			o.error(err)
			return true
		}
		if err := wm.Msg(func(b []byte) error {
			return c.WriteMessage(wm.ty, b)
		}); err != nil {
			o.error(err)
			return true
		}
		if wm.ty == websocket.PingMessage {
			time.AfterFunc(time.Duration(o.RTOMs*int(time.Millisecond)), func() {
				if time.Now().UnixMilli() > o.pingT+int64(o.RTOMs) {
					o.error(errors.New("PongFail"))
					o.Close()
				}
			})
		}
		return false
	})

	// close
	o.msg.Pull_tag_only(`close`, func(_ *WsMsg) (disable bool) {
		defer wlock.Lock()()
		if err := c.SetWriteDeadline(time.Now().Add(time.Duration(o.WTOMs * int(time.Millisecond)))); err != nil {
			o.error(err)
		}
		if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, o.Msg_normal_close)); err != nil {
			o.error(err)
		}
		return true
	})

	return o.msg, nil
}

func (o *Client) Heartbeat() (err error) {
	o.msg.Push_tag(`send`, &WsMsg{
		ty: websocket.PingMessage,
		Msg: func(f func([]byte) error) error {
			f(o.Ping.Msg)
			return nil
		},
	})
	time.Sleep(time.Duration((o.RTOMs + 100) * int(time.Millisecond)))
	return o.Error()
}

func (o *Client) Close() {
	o.msg.Push_tag(`close`, nil)
}

func (o *Client) Isclose() (isclose bool) {
	o.l.RLock()
	defer o.l.RUnlock()
	return o.close
}

func (o *Client) Error() (e error) {
	o.l.RLock()
	defer o.l.RUnlock()
	return o.err
}

func (o *Client) error(e error) {
	o.l.Lock()
	defer o.l.Unlock()
	o.err = errors.Join(o.err, e)
}
