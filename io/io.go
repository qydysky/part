package part

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// no close rc any time
// you can close wc, r, w.
func RW2Chan(r io.ReadCloser, w io.WriteCloser) (rc, wc chan []byte) {
	if r != nil {
		rc = make(chan []byte, 10)
		go func(rc chan []byte, r io.ReadCloser) {
			buf := make([]byte, 1<<16)
			for {
				n, e := r.Read(buf)
				if n != 0 {
					rc <- buf[:n]
				} else if e != nil {
					close(rc)
					break
				}
			}
		}(rc, r)
	}

	if w != nil {
		wc = make(chan []byte, 10)
		go func(wc chan []byte, w io.WriteCloser) {
			for {
				buf := <-wc
				if len(buf) == 0 { //chan close
					w.Close()
					break
				}
				_, e := w.Write(buf)
				if e != nil {
					close(wc)
					break
				}
			}
		}(wc, w)
	}
	return
}

type RWC struct {
	R func(p []byte) (n int, err error)
	W func(p []byte) (n int, err error)
	C func() error
}

func (t RWC) Write(p []byte) (n int, err error) {
	return t.W(p)
}
func (t RWC) Read(p []byte) (n int, err error) {
	return t.R(p)
}
func (t RWC) Close() error {
	return t.C()
}

func WithCtxTO(ctx context.Context, callTree string, to time.Duration, w io.Writer, r io.Reader, panicf ...func(s string)) io.ReadWriteCloser {
	var chanw atomic.Int64
	chanw.Store(time.Now().Unix())
	if len(panicf) == 0 {
		panicf = append(panicf, func(s string) { panic(s) })
	}

	go func() {
		var timer = time.NewTicker(to)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				if old, now := chanw.Load(), time.Now(); old > 0 && now.Unix()-old > int64(to.Seconds()) {
					if old != 0 {
						panicf[0](fmt.Sprintf("write blocking while close %vs > %v, goruntime leak \n%v", now.Unix()-old, to, callTree))
					}
				} else if old < 0 {
					return
				} else {
					time.AfterFunc(to, func() {
						if old, now := chanw.Load(), time.Now(); old != 0 && now.Unix()-old > int64(to.Seconds()) {
							panicf[0](fmt.Sprintf("write blocking after close %vs > %v, goruntime leak \n%v", now.Unix()-old, to, callTree))
						}
					})
				}
				return
			case now := <-timer.C:
				if old := chanw.Load(); old > 0 && now.Unix()-old > int64(to.Seconds()) {
					panicf[0](fmt.Sprintf("write blocking after rw %vs > %v, goruntime leak \n%v", now.Unix()-old, to, callTree))
					return
				} else if old < 0 {
					return
				}
			}
		}
	}()

	return RWC{
		func(p []byte) (n int, err error) {
			if n, err = r.Read(p); n != 0 {
				select {
				case <-ctx.Done():
				default:
					chanw.Store(time.Now().Unix())
				}
			}
			return
		},
		func(p []byte) (n int, err error) {
			if n, err = w.Write(p); n != 0 {
				chanw.Store(time.Now().Unix())
			}
			return
		},
		func() error {
			chanw.Store(-1)
			return nil
		},
	}
}
