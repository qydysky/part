package part

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/gorilla/websocket"

	s "github.com/qydysky/part/signal"
)

type Client struct {
	Url      string
	SendChan chan interface{}
	RecvChan chan []byte

	TO     int
	Header map[string]string
	Proxy  string

	Ping Ping

	Msg_normal_close  string
	Func_normal_close func()
	Func_abort_close  func()

	err    error
	signal *s.Signal
}

type ws_msg struct {
	Type int
	Msg  []byte
}

type Ping struct {
	Msg      []byte
	Period   int
	had_pong bool
}

func New_client(config Client) (o *Client) {
	tmp := Client{
		TO:                300 * 1000,
		Func_normal_close: func() {},
		Func_abort_close:  func() {},
		SendChan:          make(chan interface{}, 1e4),
		RecvChan:          make(chan []byte, 1e4),
	}
	tmp.Url = config.Url
	if v := config.TO; v != 0 {
		tmp.TO = v
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
	return &tmp
}

func (i *Client) Handle() (o *Client) {
	o = i

	if o.signal.Islive() {
		return
	}
	o.signal = s.Init()

	if o.Url == "" {
		o.signal.Done()
		o.err = errors.New(`url == ""`)
		return
	}

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
		o.signal.Done()
		e := err.Error()
		if response != nil {
			if response.Status != "" {
				e += ` ` + response.Status
			}
			if response.Body != nil {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					o.err = err
					return
				}
				response.Body.Close()
				e += ` ` + string(body)
			}
		}
		o.err = errors.New(e)
		return
	}

	go func() {
		defer func() {
			o.signal.Done()
			close(o.RecvChan)
			c.Close()
		}()

		done := s.Init()
		defer done.Done()

		go func() {
			defer done.Done()

			for {
				c.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(o.TO)))
				msg_type, message, err := c.ReadMessage()
				if err != nil {
					if e, ok := err.(*websocket.CloseError); ok {
						switch e.Code {
						case websocket.CloseNormalClosure:
							o.Func_normal_close()
						case websocket.CloseAbnormalClosure:
							o.Func_abort_close()
						default:
						}
						o.err = e
					}
					return
				}
				if !done.Islive() {
					return
				}
				switch msg_type {
				case websocket.PingMessage:
					o.SendChan <- ws_msg{
						Type: websocket.PongMessage,
						Msg:  message,
					}
				case websocket.PongMessage:
					o.Ping.had_pong = true
				default:
					o.RecvChan <- message
				}
			}
		}()

		for {
			select {
			case <-done.WaitC():
				return
			case t := <-o.SendChan:
				if !done.Islive() {
					return
				}

				var err error
				switch reflect.ValueOf(t).Type().Name() {
				case `ws_msg`:
					err = c.WriteMessage(t.(ws_msg).Type, t.(ws_msg).Msg)
				default:
					err = c.WriteMessage(websocket.TextMessage, t.([]byte))
				}
				if err != nil {
					o.err = err
					return
				}
				c.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(o.TO)))
			case <-o.signal.WaitC():
				if !done.Islive() {
					return
				}

				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, o.Msg_normal_close))
				if err != nil {
					o.err = err
				}
				select {
				case <-done.WaitC():
				case <-time.After(time.Second):
				}
				return
			}
		}
	}()
	return
}

func (o *Client) Heartbeat() (err error) {
	if !o.signal.Islive() {
		return errors.New(`not alive`)
	}

	var ticker_ping = time.NewTicker(time.Duration(o.TO) * time.Millisecond)
	if o.Ping.Period > 0 {
		if o.Ping.Period < o.TO {
			ticker_ping.Reset(time.Duration(o.Ping.Period) * time.Millisecond)
			o.Ping.had_pong = true
		} else {
			err = errors.New(`Ping.Period < o.TO`)
		}
	} else {
		ticker_ping.Stop()
	}

	go func(ticker_ping *time.Ticker) {
		defer ticker_ping.Stop()
		for {
			select {
			case <-ticker_ping.C:
				if !o.Ping.had_pong {
					o.err = errors.New(`Pong fail!`)
					o.Close()
					return
				}
				o.SendChan <- ws_msg{
					Type: websocket.PingMessage,
					Msg:  o.Ping.Msg,
				}
				o.Ping.had_pong = false
			case <-o.signal.Chan:
				return
			}
		}
	}(ticker_ping)

	return
}

func (o *Client) Close() {
	o.signal.Done()
}

func (o *Client) Isclose() bool {
	return !o.signal.Islive()
}

func (o *Client) Error() error {
	return o.err
}
