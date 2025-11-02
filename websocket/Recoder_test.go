package part

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	web "github.com/qydysky/part/web"
)

func Test_Play(t *testing.T) {
	t.Parallel()

	s, cf := Play("1.csv")
	defer cf()

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
		Func_normal_close: func() {
			fmt.Println(`Func_normal_close`)
		},
		Func_abort_close: func() {
			fmt.Println(`Func_abort_close`)
		},
	}); e != nil {
		t.Fatal(e)
	} else if handler, e := c.Handle(); e != nil {
		t.Fatal(e)
	} else {
		now := time.Now()
		cc := 0
		handler.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				cc += 1
				t.Log(time.Since(now), string(b))
				if bytes.Contains(b, []byte("宫本麦狗: 好耶")) {
					if math.Abs(time.Since(now).Seconds()-1.898631) > 1 {
						t.Fatal(string(b))
					}
				}
				if bytes.Contains(b, []byte("-夜未明-: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-2.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武: ybb")) {
					if math.Abs(time.Since(now).Seconds()-3.213335) > 1 {
						t.Fatal()
					}
				}
				return nil
			})
			return false
		})
		cancel, c := handler.Pull_tag_chan(`exit`, 1, context.Background())
		<-c
		cancel()
		if cc != 3 {
			t.Fatal()
		}
		t.Log("exit")
	}
}

func Test_Plays(t *testing.T) {
	t.Parallel()

	s, close := Plays(func(reg func(filepath string, start, dur time.Duration) error) {
		reg("1.csv", 0, 4*time.Second)
		reg("2.csv", 0, 4*time.Second)
	})
	defer close()

	w := web.Easy_boot()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET")
				return
			}
			conn := s.WS(w, r)
			<-conn
			<-conn
		},
	})

	// wait
	time.Sleep(time.Second)

	if c, e := New_client(&Client{
		Url: `ws://` + w.Server.Addr + `/ws`,
		Func_normal_close: func() {
			fmt.Println(`Func_normal_close`)
		},
		Func_abort_close: func() {
			fmt.Println(2)
		},
	}); e != nil {
		t.Fatal(e)
	} else if handler, e := c.Handle(); e != nil {
		t.Fatal(e)
	} else {
		now := time.Now()
		handler.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				t.Log(time.Since(now), string(b))
				if bytes.Contains(b, []byte("宫本麦狗: 好耶")) {
					if math.Abs(time.Since(now).Seconds()-1.898631) > 1 {
						t.Fatal(string(b))
					}
				}
				if bytes.Contains(b, []byte("-夜未明-: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-2.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武: ybb")) {
					if math.Abs(time.Since(now).Seconds()-3.213335) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("宫本麦狗1: 好耶")) {
					if math.Abs(time.Since(now).Seconds()-5.898631) > 1 {
						t.Fatal(string(b), time.Since(now).Seconds())
					}
				}
				if bytes.Contains(b, []byte("-夜未明-1: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-6.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武1: ybb")) {
					if math.Abs(time.Since(now).Seconds()-7.213335) > 1 {
						t.Fatal()
					}
				}
				return nil
			})
			return false
		})
		cancel, c := handler.Pull_tag_chan(`exit`, 1, context.Background())
		<-c
		cancel()
		t.Log("exit")
	}
}

func Test_PlaysSeed(t *testing.T) {
	t.Parallel()

	s, close := Plays(func(reg func(filepath string, start, dur time.Duration) error) {
		reg("1.csv", 0, 4*time.Second)
		reg("2.csv", 0, 4*time.Second)
	})
	defer close()

	w := web.Easy_boot()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET")
				return
			}
			conn := s.WS(w, r)
			<-conn
			<-conn
		},
	})

	// wait
	time.Sleep(time.Second)

	if c, e := New_client(&Client{
		Url: `ws://` + w.Server.Addr + `/ws`,
		Func_normal_close: func() {
			fmt.Println(`Func_normal_close`)
		},
		Func_abort_close: func() {
			fmt.Println(2)
		},
	}); e != nil {
		t.Fatal(e)
	} else if handler, e := c.Handle(); e != nil {
		t.Fatal(e)
	} else {
		handler.Push_tag(`send`, &WsMsg{
			Msg: func(f func([]byte) error) error {
				f([]byte("3"))
				return nil
			},
		})
		now := time.Now()
		cc := 0
		handler.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				t.Log(time.Since(now), string(b))
				cc += 1
				if bytes.Contains(b, []byte("宫本麦狗: 好耶")) {
					t.Fatal(string(b))
				}
				if bytes.Contains(b, []byte("-夜未明-: 七点二十要做核酸，干脆不睡了吧")) {
					t.Fatal()
				}
				if bytes.Contains(b, []byte("信长の天下布武: ybb")) {
					if math.Abs(time.Since(now).Seconds()-0.213335) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("宫本麦狗1: 好耶")) {
					if math.Abs(time.Since(now).Seconds()-2.898631) > 1 {
						t.Fatal(string(b), time.Since(now).Seconds())
					}
				}
				if bytes.Contains(b, []byte("-夜未明-1: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-3.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武1: ybb")) {
					if math.Abs(time.Since(now).Seconds()-4.213335) > 1 {
						t.Fatal()
					}
				}
				return nil
			})
			return false
		})
		cancel, c := handler.Pull_tag_chan(`exit`, 1, context.Background())
		<-c
		cancel()
		if cc != 4 {
			t.Fatal()
		}
		t.Log("exit")
	}
}

func Test_PlaysStart(t *testing.T) {
	t.Parallel()

	s, close := Plays(func(reg func(filepath string, start, dur time.Duration) error) {
		reg("1.csv", 2*time.Second, 4*time.Second)
		reg("2.csv", 0, 4*time.Second)
	})
	defer close()

	w := web.Easy_boot()
	w.Handle(map[string]func(http.ResponseWriter, *http.Request){
		`/ws`: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Allow", "GET")
				return
			}
			conn := s.WS(w, r)
			<-conn
			<-conn
		},
	})

	// wait
	time.Sleep(time.Second)

	if c, e := New_client(&Client{
		Url: `ws://` + w.Server.Addr + `/ws`,
		Func_normal_close: func() {
			fmt.Println(`Func_normal_close`)
		},
		Func_abort_close: func() {
			fmt.Println(2)
		},
	}); e != nil {
		t.Fatal(e)
	} else if handler, e := c.Handle(); e != nil {
		t.Fatal(e)
	} else {
		now := time.Now()
		cc := 0
		handler.Pull_tag_only(`recv`, func(wm *WsMsg) (disable bool) {
			wm.Msg(func(b []byte) error {
				t.Log(time.Since(now), string(b))
				cc += 1
				if bytes.Contains(b, []byte("宫本麦狗: 好耶")) {
					t.Fatal(string(b))
				}
				if bytes.Contains(b, []byte("-夜未明-: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-0.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武: ybb")) {
					if math.Abs(time.Since(now).Seconds()-1.213335) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("宫本麦狗1: 好耶")) {
					if math.Abs(time.Since(now).Seconds()-3.898631) > 1 {
						t.Fatal(string(b), time.Since(now).Seconds())
					}
				}
				if bytes.Contains(b, []byte("-夜未明-1: 七点二十要做核酸，干脆不睡了吧")) {
					if math.Abs(time.Since(now).Seconds()-4.360965) > 1 {
						t.Fatal()
					}
				}
				if bytes.Contains(b, []byte("信长の天下布武1: ybb")) {
					if math.Abs(time.Since(now).Seconds()-5.213335) > 1 {
						t.Fatal()
					}
				}
				return nil
			})
			return false
		})
		cancel, c := handler.Pull_tag_chan(`exit`, 1, context.Background())
		<-c
		cancel()
		if cc != 5 {
			t.Fatal(cc)
		}
		t.Log("exit")
	}
}
