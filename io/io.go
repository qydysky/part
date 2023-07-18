package part

import (
	"context"
	"errors"
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
	if t.W != nil {
		return t.W(p)
	}
	return 0, nil
}
func (t RWC) Read(p []byte) (n int, err error) {
	if t.R != nil {
		return t.R(p)
	}
	return 0, nil
}
func (t RWC) Close() error {
	if t.C != nil {
		return t.C()
	}
	return nil
}

// close reader by yourself
// call Close() after writer fin
func WithCtxTO(ctx context.Context, callTree string, to time.Duration, w []io.WriteCloser, r io.Reader, panicf ...func(s string)) io.ReadWriteCloser {
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
				// avoid write block
				for i := 0; i < len(w); i++ {
					w[i].Close()
				}
				if old, now := chanw.Load(), time.Now(); old > 0 && now.Unix()-old > int64(to.Seconds()) {
					panicf[0](fmt.Sprintf("rw blocking while close %vs > %v, goruntime leak \n%v", now.Unix()-old, to, callTree))
				} else {
					time.AfterFunc(to, func() {
						if chanw.Load() != -1 {
							panicf[0](fmt.Sprintf("rw blocking after close %v, goruntime leak \n%v", to, callTree))
						}
					})
				}
				return
			case now := <-timer.C:
				if old := chanw.Load(); old > 0 && now.Unix()-old > int64(to.Seconds()) {
					panicf[0](fmt.Sprintf("rw blocking after rw %vs > %v, goruntime leak \n%v", now.Unix()-old, to, callTree))
					return
				}
			}
		}
	}()

	return RWC{
		func(p []byte) (n int, err error) {
			select {
			case <-ctx.Done():
				err = context.Canceled
			default:
				if n, err = r.Read(p); n != 0 {
					chanw.Store(time.Now().Unix())
				}
			}
			return
		},
		func(p []byte) (n int, err error) {
			select {
			case <-ctx.Done():
				err = context.Canceled
			default:
				for i := 0; i < len(w); i++ {
					if n, err = w[i].Write(p); n != 0 {
						chanw.Store(time.Now().Unix())
					}
				}
			}
			return
		},
		func() error {
			chanw.Store(-1)
			return nil
		},
	}
}

var (
	ErrWrite = errors.New("ErrWrite")
	ErrRead  = errors.New("ErrRead")
)

// close reader by yourself
func WithCtxCopy(ctx context.Context, callTree string, to time.Duration, w []io.WriteCloser, r io.Reader, panicf ...func(s string)) error {
	rwc := WithCtxTO(ctx, callTree, to, w, r)
	defer rwc.Close()
	for buf := make([]byte, 2048); true; {
		if n, e := rwc.Read(buf); n != 0 {
			if n, e := rwc.Write(buf[:n]); n == 0 && e != nil {
				if !errors.Is(e, io.EOF) {
					return errors.Join(ErrWrite, e)
				}
				break
			}
		} else if e != nil {
			if !errors.Is(e, io.EOF) {
				return errors.Join(ErrRead, e)
			}
			break
		}
	}
	return nil
}
