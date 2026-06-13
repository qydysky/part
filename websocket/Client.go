package part

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
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

	// RTOMs int           // default: 300s
	// WTOMs int           // default: 300s
	RTO time.Duration // default: 300s
	WTO time.Duration // default: 300s
	CTO time.Duration // default: 5s
	// BufSize int               // msg buf 1: always use single buf >1: use bufs cycle. default:10
	Header map[string]string // default: map[string]string{}
	Proxy  string

	Ping      Ping // default: no ping
	pingT     atomic.Pointer[time.Timer]
	closeCall atomic.Bool

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
	Msg []byte
	// Period    int // ms
	PeriodDur time.Duration
}

func NewClient(config *Client) (*Client, error) {
	tmp := Client{
		RTO:               300 * time.Second,
		WTO:               300 * time.Second,
		CTO:               5 * time.Second,
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
	if config.RTO != 0 {
		tmp.RTO = config.RTO
	}
	if config.WTO != 0 {
		tmp.WTO = config.WTO
	}
	// if tmp.RTO == 0 {
	// 	tmp.RTO = time.Duration(tmp.RTOMs * int(time.Millisecond))
	// }
	// if tmp.WTO == 0 {
	// 	tmp.WTO = time.Duration(tmp.WTOMs * int(time.Millisecond))
	// }
	if config.CTO != 0 {
		tmp.CTO = config.CTO
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
	// if config.Ping.Period != 0 {
	// 	tmp.Ping = config.Ping
	// }
	if config.Ping.PeriodDur != 0 {
		tmp.Ping.PeriodDur = config.Ping.PeriodDur
	}
	return &tmp, nil
}

// 处理
//
// // 发送数据到服务器
//
//	msg.Push_tag(`send`, &WsMsg{
//		Msg: func(f func([]byte) error) error {
//			return f([]byte("test"))
//		},
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
	} else if err := c.SetWriteDeadline(time.Now().Add(o.WTO)); err != nil {
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
			if e := c.SetReadDeadline(time.Now().Add(o.RTO)); e != nil {
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
						if ti := o.pingT.Swap(nil); ti != nil {
							ti.Stop()
						}
						if o.Ping.PeriodDur != 0 {
							time.AfterFunc(o.Ping.PeriodDur, func() {
								o.msg.Push_tag(`send`, &WsMsg{
									ty: websocket.PingMessage,
									Msg: func(f func([]byte) error) error {
										return f(o.Ping.Msg)
									},
								})
							})
						}
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
			if o.closeCall.Load() && errors.Is(err, net.ErrClosed) {
				err = &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: err.Error()}
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
		if err := wm.Msg(func(b []byte) error {
			switch wm.ty {
			case websocket.CloseMessage:
				o.closeCall.Store(true)
				time.AfterFunc(o.CTO, func() {
					c.Close()
				})
				return c.WriteControl(wm.ty, b, time.Now().Add(o.WTO))
			case websocket.PingMessage:
				o.pingT.Store(time.AfterFunc(o.RTO, func() {
					o.error(errors.New("PongFail"))
					o.closeCall.Store(true)
					time.AfterFunc(o.CTO, func() {
						c.Close()
					})
					if e := c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, o.Msg_normal_close), time.Now().Add(o.WTO)); e != nil {
						o.error(e)
					}
				}))
				return c.WriteControl(wm.ty, b, time.Now().Add(o.WTO))
			case websocket.PongMessage:
				return c.WriteControl(wm.ty, b, time.Now().Add(o.WTO))
			default:
				if err := c.SetWriteDeadline(time.Now().Add(o.WTO)); err != nil {
					o.error(err)
				}
				return c.WriteMessage(wm.ty, b)
			}
		}); err != nil {
			o.error(err)
			return true
		}
		return false
	})

	return o.msg, nil
}

func (o *Client) Heartbeat() (err error) {
	o.msg.Push_tag(`send`, &WsMsg{
		ty: websocket.PingMessage,
		Msg: func(f func([]byte) error) error {
			return f(o.Ping.Msg)
		},
	})
	return o.Error()
}

func (o *Client) Close() {
	o.msg.Push_tag(`send`, &WsMsg{
		ty: websocket.CloseMessage,
		Msg: func(f func([]byte) error) error {
			return f(websocket.FormatCloseMessage(websocket.CloseNormalClosure, o.Msg_normal_close))
		},
	})
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
