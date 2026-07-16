package part

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	pc "github.com/qydysky/part/ctx"
)

type Buf[T any] struct {
	maxsize  int
	bufsize  int
	modified Modified
	buf      []T
	l        sync.RWMutex
}

type Modified struct {
	p uintptr
	t uint64
}

// normally, use GetLock() or GetRLock() in mutil goroutines
//
// or use other export methods without lock
func New[T any](maxsize ...int) *Buf[T] {
	t := new(Buf[T])
	if len(maxsize) > 0 {
		t.maxsize = maxsize[0]
	}
	t.modified.p = uintptr(unsafe.Pointer(t))
	return t
}

func (t *Buf[T]) Clear() {
	t.buf = nil
	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) Size() int {
	return t.bufsize
}

// unsafe without lock
func (t *Buf[T]) SetSize(n int) {
	t.bufsize = n
	t.resetSize()
	t.modified.t += 1
}

// unsafe without lock
func (t *Buf[T]) AddSize(n int) {
	t.bufsize += n
	t.resetSize()
	t.modified.t += 1
}

func (t *Buf[T]) resetSize() {
	if cap(t.buf) < t.bufsize {
		t.buf = append(t.buf[:cap(t.buf)], make([]T, t.bufsize-cap(t.buf))...)
	} else if len(t.buf) < t.bufsize {
		t.buf = t.buf[:t.bufsize]
	}
}

func (t *Buf[T]) IsEmpty() bool {
	return t.bufsize == 0
}

func (t *Buf[T]) Reset() {
	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) AppendTo(to *Buf[T]) error {
	return to.Append(t.GetPureBuf())
}

var ErrOverMax = errors.New("slices.Append.ErrOverMax")

func (t *Buf[T]) Cap() int {
	return cap(t.buf)
}

// may alloc more mem
func (t *Buf[T]) ExpandCapTo(size int) {
	if cap(t.buf) >= size {
		return
	} else {
		t.buf = append(t.buf[:cap(t.buf)], make([]T, size-cap(t.buf))...)
	}
}

// may alloc more mem
func (t *Buf[T]) ExpandCap(size int) {
	t.buf = append(t.buf[:cap(t.buf)], make([]T, size)...)
}

func (t *Buf[T]) Append(data []T) error {
	if t.buf == nil {
		t.buf = make([]T, len(data))
	} else if t.maxsize != 0 && t.bufsize+len(data) > t.maxsize {
		return ErrOverMax
	}
	t.buf = append(t.buf[:t.bufsize], data...)
	t.bufsize += len(data)
	t.modified.t += 1
	return nil
}

type BufAppendsI[T any] struct {
	bufp *Buf[T]
	e    error
}

func (t *BufAppendsI[T]) Append(data []T) *BufAppendsI[T] {
	if t.e != nil || len(data) == 0 {
		return t
	}
	t.e = t.bufp.Append(data)
	return t
}

func (t *Buf[T]) Appends(f func(ba *BufAppendsI[T])) error {
	ba := &BufAppendsI[T]{bufp: t}
	f(ba)
	return ba.e
}

func (t *Buf[T]) SetFrom(data []T) error {
	if t.buf == nil {
		t.buf = make([]T, len(data))
	} else if t.maxsize != 0 && t.bufsize+len(data) > t.maxsize {
		return ErrOverMax
	}
	t.buf = append(t.buf[:0], data...)
	t.bufsize = len(data)
	t.modified.t += 1
	return nil
}

var ErrOverLen = errors.New("slices.Remove.ErrOverLen")

func (t *Buf[T]) RemoveFront(n int) error {
	if n <= 0 {
		return nil
	}
	if t.bufsize < n {
		return ErrOverLen
	} else if t.bufsize == n {
		t.bufsize = 0
	} else {
		t.resetSize()
		t.bufsize = copy(t.buf, t.buf[n:t.bufsize])
	}

	t.modified.t += 1
	return nil
}

func (t *Buf[T]) RemoveBack(n int) error {
	if n <= 0 {
		return nil
	}
	if t.bufsize < n {
		return ErrOverLen
	} else if t.bufsize == n {
		t.bufsize = 0
	} else {
		t.bufsize -= n
	}

	t.modified.t += 1
	return nil
}

// unsafe without lock
func (t *Buf[T]) SetModified() {
	t.modified.t += 1
}

func (t *Buf[T]) GetModified() Modified {
	return t.modified
}

var ErrNoSameBuf = errors.New("slices.HadModified.ErrNoSameBuf")

func (t *Buf[T]) HadModified(mt Modified) (modified bool, err error) {
	if t.modified.p != mt.p {
		err = ErrNoSameBuf
	}
	modified = t.modified.t != mt.t
	return
}

// unsafe without lock
func (t *Buf[T]) GetRawBuf(op, ed int) (buf []T) {
	return t.buf[op:ed]
}

// unsafe without lock
func (t *Buf[T]) GetPureBuf() (buf []T) {
	return t.buf[:t.bufsize]
}

func (t *Buf[T]) GetCopyBuf() (buf []T) {
	buf = make([]T, t.bufsize)
	copy(buf, t.buf[:t.bufsize])
	return
}

type BufIOM[T any] struct {
	ctx    context.Context
	closed atomic.Bool
	t      *Buf[T]
}

func (t *Buf[T]) IO() *BufIOM[T] {
	return &BufIOM[T]{ctx: context.Background(), t: t}
}

func (t *BufIOM[T]) Ctx(ctx context.Context) *BufIOM[T] {
	t.t.l.Lock()
	t.ctx = ctx
	t.t.l.Unlock()
	return t
}

func (t *BufIOM[T]) Read(b []T) (n int, e error) {
	for {
		t.t.l.RLock()
		if t.t.bufsize > 0 {
			n = copy(b, t.t.GetPureBuf())
			_ = t.t.RemoveFront(n)
			if t.t.bufsize == 0 && t.closed.Load() {
				e = io.EOF
			}
			t.t.l.RUnlock()
			return
		} else {
			t.t.l.RUnlock()
			select {
			case <-t.ctx.Done():
				e = context.Canceled
				return
			case <-time.After(time.Millisecond * 100):
			}
		}
	}
}

func (t *BufIOM[T]) ReadFrom(r interface {
	Read(p []T) (n int, err error)
}) (total int64, e error) {
	t.t.l.Lock()
	defer t.t.l.Unlock()
	return t.t.ReadFrom(r)
}

func (t *BufIOM[T]) WriteTo(w interface {
	Write(p []T) (n int, err error)
}) (total int64, e error) {
	t.t.l.RLock()
	defer t.t.l.RUnlock()
	return t.t.WriteTo(w)
}

func (t *BufIOM[T]) Write(b []T) (n int, e error) {
	if t.closed.Load() {
		e = io.ErrClosedPipe
		return
	}
	if pc.Done(t.ctx) {
		e = context.Canceled
		return
	}
	t.t.l.Lock()
	defer t.t.l.Unlock()
	return len(b), t.t.Append(b)
}

func (t *BufIOM[T]) Close() (e error) {
	t.closed.Store(true)
	return nil
}

func (t *BufIOM[T]) GetLock() (i interface {
	Unlock()
	BufLockMI[T]
}) {
	t.t.l.Lock()
	return &BufLockM[T]{t.t}
}

func (t *BufIOM[T]) GetRLock() (i interface {
	RUnlock()
	BufRLockMI[T]
}) {
	t.t.l.RLock()
	return &BufRLockM[T]{t.t}
}

var _ io.ReadWriteCloser = New[byte]().IO()

// func (t *Buf[T]) Read(b []T) (n int, e error) {
// 	for {
// 		time.Sleep(time.Millisecond * 100)
// 		t.l.RLock()
// 		if t.Size() > 0 {
// 			n = copy(b, t.GetPureBuf())
// 			e = t.RemoveFront(n)
// 			t.l.RUnlock()
// 			return
// 		} else {
// 			t.l.RUnlock()
// 		}
// 	}
// }

// func (t *Buf[T]) Write(b []T) (n int, e error) {
// 	t.l.Lock()
// 	defer t.l.Unlock()
// 	return len(b), t.Append(b)
// }

type BufRLockMI[T any] interface {
	IsEmpty() bool
	Size() int
	Cap() int
	// Read(b []T) (n int, e error)
	RemoveFront(int) error
	RemoveBack(int) error
	AppendTo(to *Buf[T]) error
	GetCopyBuf() []T
	GetPureBuf() []T          // unsafe
	GetRawBuf(op, ed int) []T // unsafe
	GetModified() Modified
	HadModified(mt Modified) (modified bool, err error)
}

type BufLockMI[T any] interface {
	Clear()
	Reset()
	ExpandCapTo(size int)
	// Write(b []T) (n int, e error)
	ReadFrom(r interface {
		Read(p []T) (n int, err error)
	}) (int64, error)
	Append(data []T) error
	Appends(f func(ba *BufAppendsI[T])) error
	SetFrom(data []T) error
	RemoveFront(n int) error
	RemoveBack(n int) error
	SetModified()
	SetSize(int)
	BufRLockMI[T]
}

type BufLockM[T any] struct {
	*Buf[T]
}

func (b *BufLockM[T]) Unlock() {
	b.l.Unlock()
}

func (t *Buf[T]) GetLock() (i interface {
	Unlock()
	BufLockMI[T]
}) {
	t.l.Lock()
	return &BufLockM[T]{t}
}

type BufRLockM[T any] struct {
	*Buf[T]
}

func (b *BufRLockM[T]) RUnlock() {
	b.l.RUnlock()
}

func (t *Buf[T]) GetRLock() (i interface {
	RUnlock()
	BufRLockMI[T]
}) {
	t.l.RLock()
	return &BufRLockM[T]{t}
}

// ReadFrom will reset buf first
func (t *Buf[T]) ReadFrom(r interface {
	Read(p []T) (n int, err error)
}) (total int64, e error) {
	t.Reset()
	for {
		if cap(t.buf) == t.bufsize {
			t.buf = append(t.buf, *new(T))
			t.buf = t.buf[:cap(t.buf)]
		}
		if n, err := r.Read(t.buf[t.bufsize:cap(t.buf)]); n > 0 {
			t.bufsize += n
			total += int64(n)
			t.modified.t += 1
		} else if err != nil {
			if !errors.Is(err, io.EOF) {
				e = err
			}
			return
		}
	}
}

// ReadN will reset buf first
func (t *Buf[T]) ReadN(r interface {
	Read(p []T) (n int, err error)
}, n int) (total int64, e error) {
	t.Reset()
	return t.ReadMoreN(r, n)
}

func (t *Buf[T]) ReadMoreN(r interface {
	Read(p []T) (n int, err error)
}, n int) (total int64, e error) {
	for n > 0 {
		if cap(t.buf) == t.bufsize {
			t.buf = append(t.buf, *new(T))
			t.buf = t.buf[:cap(t.buf)]
		}
		if rn, err := r.Read(t.buf[t.bufsize:min(t.bufsize+n, cap(t.buf))]); rn > 0 {
			n -= rn
			t.bufsize += rn
			total += int64(rn)
			t.modified.t += 1
		} else if err != nil {
			e = err
			return
		}
	}
	return
}

func (t *Buf[T]) WriteTo(w interface {
	Write(p []T) (n int, err error)
}) (total int64, e error) {
	for t.bufsize > 0 {
		if n, err := w.Write(t.GetPureBuf()); n > 0 {
			t.modified.t += 1
			t.bufsize -= n
			total += int64(n)
		} else if errors.Is(err, io.EOF) {
			return total, nil
		} else {
			return total, err
		}
	}
	return total, nil
}
