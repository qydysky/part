package part

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
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

func NewPipe() *IOpipe {
	r, w := io.Pipe()
	return &IOpipe{R: r, W: w}
}

type onceError struct {
	sync.Mutex // guards following
	err        error
}

func (a *onceError) Store(err error) {
	a.Lock()
	defer a.Unlock()
	if a.err != nil {
		return
	}
	a.err = err
}
func (a *onceError) Load() error {
	a.Lock()
	defer a.Unlock()
	return a.err
}

type IOpipe struct {
	R  *io.PipeReader
	W  *io.PipeWriter
	re onceError
	we onceError
}

func (t *IOpipe) Write(p []byte) (n int, err error) {
	if t.W != nil {
		n, err = t.W.Write(p)
		if errors.Is(err, io.ErrClosedPipe) {
			err = errors.Join(err, t.we.Load())
		}
	}
	return
}
func (t *IOpipe) Read(p []byte) (n int, err error) {
	if t.R != nil {
		n, err = t.R.Read(p)
		if errors.Is(err, io.ErrClosedPipe) {
			err = errors.Join(err, t.re.Load())
		}
	}
	return
}
func (t *IOpipe) Close() (err error) {
	if t.W != nil {
		err = errors.Join(err, t.W.Close())
	}
	if t.R != nil {
		err = errors.Join(err, t.R.Close())
	}
	return
}
func (t *IOpipe) CloseWithError(e error) (err error) {
	if t.W != nil {
		t.we.Store(e)
		err = errors.Join(err, t.W.CloseWithError(e))
	}
	if t.R != nil {
		t.re.Store(e)
		err = errors.Join(err, t.R.CloseWithError(e))
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
//
// to avoid writer block after ctx done, you should close writer after ctx done
//
// call Close() after writer fin
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
				_, err = w.Write(p)
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

var (
	ErrWrite   = errors.New("ErrWrite")
	ErrRead    = errors.New("ErrRead")
	ErrLoopMax = errors.New("ErrLoopMax")
)

// close reader by yourself
//
// to avoid writer block after ctx done, you should close writer after ctx done
//
// call Close() after writer fin
func WithCtxCopy(ctx context.Context, callTree string, copybuf []byte, to time.Duration, w io.Writer, r io.Reader, panicf ...func(s string)) error {
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

	defer chanw.Store(-1)

	for {
		select {
		case <-ctx.Done():
			return errors.Join(ErrRead, context.Canceled)
		default:
			n, e := r.Read(copybuf)
			chanw.Store(time.Now().Unix())
			if n != 0 {
				select {
				case <-ctx.Done():
					return errors.Join(ErrRead, context.Canceled)
				default:
					n, e := w.Write(copybuf[:n])
					chanw.Store(time.Now().Unix())
					if n == 0 && e != nil {
						if !errors.Is(e, io.EOF) {
							return errors.Join(ErrWrite, e)
						}
						return nil
					}
				}
			} else if e != nil {
				if !errors.Is(e, io.EOF) {
					return errors.Join(ErrRead, e)
				}
				return nil
			}
		}
	}
}

type CopyConfig struct {
	BytePerLoop, MaxLoop, MaxByte, BytePerSec uint64
	SkipByte                                  int
}

// close by yourself
//
// watch out uint64(c.MaxLoop*c.BytePerLoop) overflow
func Copy(r io.Reader, w io.Writer, c CopyConfig) (e error) {
	var (
		ticker *time.Ticker
		leftN  uint64
	)
	if c.BytePerSec > 0 {
		if c.BytePerLoop == 0 || c.BytePerLoop > c.BytePerSec {
			c.BytePerLoop = c.BytePerSec
		}
		ticker = time.NewTicker(time.Second)
		defer ticker.Stop()
	}
	if c.BytePerLoop == 0 {
		c.BytePerLoop = 1 << 17
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	} else if c.BytePerLoop > 1<<17 {
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.BytePerLoop = 1 << 17
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.MaxByte = c.MaxLoop * c.BytePerLoop
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	} else {
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.MaxByte = c.MaxLoop * c.BytePerLoop
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	}

	if seeker, ok := r.(io.Seeker); ok && c.SkipByte > 0 {
		_, e = seeker.Seek(int64(c.SkipByte), io.SeekCurrent)
		if e != nil {
			return
		}
		c.SkipByte = 0
	}

	buf := make([]byte, c.BytePerLoop)
	readC := uint64(0)
	for {
		if n, err := r.Read(buf); n != 0 {
			if c.SkipByte > 0 {
				if n <= int(c.SkipByte) {
					c.SkipByte -= n
					return
				} else {
					n, e = w.Write(buf[c.SkipByte:])
					c.SkipByte = 0
				}
			} else {
				n, e = w.Write(buf[:n])
			}
			if e != nil {
				return
			}
			if c.BytePerSec != 0 {
				readC += uint64(n)
			}
		} else if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			} else {
				return nil
			}
		}

		if c.MaxLoop > 0 {
			c.MaxLoop -= 1
			if c.MaxLoop == 1 && leftN != 0 {
				buf = buf[:leftN]
			} else if c.MaxLoop == 0 {
				return ErrLoopMax
			}
		}
		if c.BytePerSec != 0 && readC >= c.BytePerSec {
			<-ticker.C
			readC = 0
		}
	}
}

func WriterWithConfig(w io.Writer, c CopyConfig) (wc io.Writer) {
	var leftN uint64
	if c.BytePerSec > 0 {
		if c.BytePerLoop == 0 || c.BytePerLoop > c.BytePerSec {
			c.BytePerLoop = c.BytePerSec
		}
	}
	if c.BytePerLoop == 0 {
		c.BytePerLoop = 1 << 17
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	} else if c.BytePerLoop > 1<<17 {
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.BytePerLoop = 1 << 17
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.MaxByte = c.MaxLoop * c.BytePerLoop
				c.BytePerLoop = 1 << 17
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	} else {
		if c.MaxLoop == 0 {
			if c.MaxByte != 0 {
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		} else {
			if c.MaxByte != 0 {
				c.MaxByte = min(c.MaxByte, c.MaxLoop*c.BytePerLoop)
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			} else {
				c.MaxByte = c.MaxLoop * c.BytePerLoop
				leftN = c.MaxByte % c.BytePerLoop
				c.MaxLoop = c.MaxByte / c.BytePerLoop
				if leftN > 0 {
					c.MaxLoop += 1
				}
			}
		}
	}
	buf := make([]byte, c.BytePerLoop)
	readC := uint64(0)

	rwc := RWC{
		W: func(p []byte) (n int, e error) {
			if c.MaxLoop > 0 {
				c.MaxLoop -= 1
				if c.MaxLoop == 1 && leftN != 0 {
					buf = buf[:leftN]
				} else if c.MaxLoop == 0 {
					return 0, ErrLoopMax
				}
			}
			if c.BytePerSec != 0 && readC >= c.BytePerSec {
				time.Sleep(time.Second)
				readC = 0
			}
			if c.SkipByte > 0 {
				if len(p) <= int(c.SkipByte) {
					c.SkipByte -= len(p)
					return
				} else {
					n, e = w.Write(p[c.SkipByte:])
					c.SkipByte = 0
				}
			} else {
				n, e = w.Write(p)
			}
			if c.BytePerSec != 0 {
				readC += uint64(n)
			}
			return
		},
	}

	return rwc
}

func ReadAll(r io.Reader, b []byte) ([]byte, error) {
	b = b[:0]
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

var (
	ErrCacheWriterBusy = errors.New(`ErrCacheWriterBusy`)
)

type CacheWriter struct {
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc
	w               io.Writer
	max             uint32
	is              []cacheWriterItem
	rc              sync.Mutex
	cc              chan uint32
	c               atomic.Uint32
}

type cacheWriterItem struct {
	buf []byte
	l   atomic.Bool
}

func NewCacheWriter(ws io.Writer, max uint32) *CacheWriter {
	t := CacheWriter{w: ws, cc: make(chan uint32, max), max: max, is: make([]cacheWriterItem, max)}
	t.ctx, t.cancelCauseFunc = context.WithCancelCause(context.Background())
	return &t
}

func (t *CacheWriter) Write(b []byte) (n int, e error) {
	select {
	case <-t.ctx.Done():
		return 0, t.ctx.Err()
	default:
	}

	i := t.c.Add(1) % t.max
	if !t.is[i].l.CompareAndSwap(false, true) {
		return 0, ErrCacheWriterBusy
	}

	t.is[i].buf = append(t.is[i].buf[:0], b...)
	t.cc <- i

	go func() {
		t.rc.Lock()
		defer t.rc.Unlock()

		i := <-t.cc
		defer t.is[i].l.Store(false)
		if _, err := t.w.Write(t.is[i].buf); err != nil {
			t.cancelCauseFunc(err)
		}
	}()
	return len(b), t.ctx.Err()
}
