package part

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	msgq "github.com/qydysky/part/msgq"
)

type Client struct {
	Url string
	// rec send close
	msg *msgq.MsgType[*WsMsg]

	TO     int // depercated: use RTOMs WTOMs instead
	RTOMs  int
	WTOMs  int
	Header map[string]string
	Proxy  string

	Ping  Ping
	pingT int64

	Msg_normal_close  string
	Func_normal_close func()
	Func_abort_close  func()

	close bool
	err   error

	l sync.RWMutex
}

type WsMsg struct {
	Type int
	Msg  []byte
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
		msg:               msgq.NewType[*WsMsg](),
	}
	tmp.Url = config.Url
	if tmp.Url == "" {
		return nil, errors.New(`url == ""`)
	}
	if v := config.TO; v != 0 {
		tmp.RTOMs = v
		tmp.WTOMs = v
	}
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
//	msg.PushLock_tag(`send`, &WsMsg{
//		Msg:  []byte("message"),
//	})
//
//	msg.Pull_tag_only(`rec`, func(wm *WsMsg) (disable bool) {
//		fmt.Println(string(wm.Msg))
//		return false
//	})
//
// 事件 send rec close exit
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
				e += ` ` + string(body)
			}
		}
		return nil, errors.New(e)
	}

	// rec
	go func() {
		defer func() {
			o.msg.PushLock_tag(`exit`, nil)
			o.l.Lock()
			o.close = true
			o.l.Unlock()
		}()

		for {
			if err := c.SetReadDeadline(time.Now().Add(time.Duration(o.RTOMs * int(time.Millisecond)))); err != nil {
				o.error(err)
				return
			}
			msg_type, message, err := c.ReadMessage()
			if err != nil {
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
				} else {
					o.error(err)
				}
				return
			}

			switch msg_type {
			case websocket.PingMessage:
				o.msg.PushLock_tag(`send`, &WsMsg{
					Type: websocket.PongMessage,
					Msg:  message,
				})
			case websocket.PongMessage:
				o.pingT = time.Now().UnixMilli()
				time.AfterFunc(time.Duration(o.Ping.Period*int(time.Millisecond)), func() {
					o.msg.PushLock_tag(`send`, &WsMsg{
						Type: websocket.PingMessage,
						Msg:  o.Ping.Msg,
					})
				})
				o.Ping.had_pong = true
			default:
				o.msg.PushLock_tag(`rec`, &WsMsg{
					Type: websocket.TextMessage,
					Msg:  message,
				})
			}
		}
	}()

	// send
	o.msg.Pull_tag_only(`send`, func(wm *WsMsg) (disable bool) {
		if wm.Type == 0 {
			wm.Type = websocket.TextMessage
		}
		if err := c.SetWriteDeadline(time.Now().Add(time.Duration(o.WTOMs * int(time.Millisecond)))); err != nil {
			o.error(err)
			return true
		}
		if err := c.WriteMessage(wm.Type, wm.Msg); err != nil {
			o.error(err)
			return true
		}
		if wm.Type == websocket.PingMessage {
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
	o.msg.PushLock_tag(`send`, &WsMsg{
		Type: websocket.PingMessage,
		Msg:  o.Ping.Msg,
	})
	time.Sleep(time.Duration((o.RTOMs + 100) * int(time.Millisecond)))
	return o.Error()
}

func (o *Client) Close() {
	o.msg.PushLock_tag(`close`, nil)
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
