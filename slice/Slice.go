package part

import (
	"errors"
	"io"
	"iter"
	"sync"
	"time"
	"unsafe"

	perrors "github.com/qydysky/part/errors"
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

var ErrOverMax = perrors.New("slices.Append", "ErrOverMax")

func (t *Buf[T]) Cap() int {
	return cap(t.buf)
}

func (t *Buf[T]) ExpandCapTo(size int) {
	if cap(t.buf) >= size {
		return
	} else {
		t.buf = append(t.buf[:cap(t.buf)], make([]T, size-cap(t.buf))...)
	}
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

var ErrOverLen = perrors.New("slices.Remove", "ErrOverLen")

func (t *Buf[T]) RemoveFront(n int) error {
	if n <= 0 {
		return nil
	}
	if t.bufsize < n {
		return ErrOverLen
	} else if t.bufsize == n {
		t.bufsize = 0
	} else {
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

// unsafe
func (t *Buf[T]) SetModified() {
	t.modified.t += 1
}

func (t *Buf[T]) GetModified() Modified {
	return t.modified
}

var ErrNoSameBuf = perrors.New("slices.HadModified", "ErrNoSameBuf")

func (t *Buf[T]) HadModified(mt Modified) (modified bool, err error) {
	if t.modified.p != mt.p {
		err = ErrNoSameBuf
	}
	modified = t.modified.t != mt.t
	return
}

// unsafe
func (t *Buf[T]) GetPureBuf() (buf []T) {
	return t.buf[:t.bufsize]
}

func (t *Buf[T]) GetCopyBuf() (buf []T) {
	buf = make([]T, t.bufsize)
	copy(buf, t.buf[:t.bufsize])
	return
}

func (t *Buf[T]) Read(b []T) (n int, e error) {
	for t.Size() == 0 {
		time.Sleep(time.Millisecond * 100)
	}
	n = copy(b, t.GetPureBuf())
	e = t.RemoveFront(n)
	return
}

func (t *Buf[T]) Write(b []T) (n int, e error) {
	return len(b), t.Append(b)
}

type BufRLockMI[T any] interface {
	IsEmpty() bool
	Size() int
	Cap() int
	Read(b []T) (n int, e error)
	GetCopyBuf() []T
	RemoveFront(int) error
	RemoveBack(int) error
	AppendTo(to *Buf[T]) error
	GetPureBuf() []T
	GetModified() Modified
	HadModified(mt Modified) (modified bool, err error)
}

type BufLockMI[T any] interface {
	Clear()
	Reset()
	ExpandCapTo(size int)
	Write(b []T) (n int, e error)
	ReadFrom(r interface {
		Read(p []T) (n int, err error)
	}) (int64, error)
	Append(data []T) error
	Appends(f func(ba *BufAppendsI[T])) error
	SetFrom(data []T) error
	RemoveFront(n int) error
	RemoveBack(n int) error
	SetModified()
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

var _ io.ReadWriter = New[byte]()

func DelFront[S ~[]T, T any](s *S, beforeIndex int) {
	*s = (*s)[:copy(*s, (*s)[beforeIndex:])]
}

func AddFront[S ~[]T, T any](s *S, t *T) {
	*s = append(*s, *new(T))
	*s = (*s)[:1+copy((*s)[1:], *s)]
	(*s)[0] = *t
}

func DelBack[S ~[]T, T any](s *S, fromIndex int) {
	*s = (*s)[:fromIndex]
}

func AddBack[S ~[]T, T any](s *S, t *T) {
	*s = append(*s, *t)
}

func LoopAddBack[S ~[]T, T any](s *S, t *T) {
	DelFront(s, 1)
	AddBack(s, t)
}

func LoopAddFront[S ~[]T, T any](s *S, t *T) {
	DelBack(s, len(*s)-1)
	AddFront(s, t)
}

func Resize[S ~[]T, T any](s *S, size int) {
	if len(*s) >= size || cap(*s) >= size {
		*s = (*s)[:size]
	} else {
		*s = append((*s)[:cap(*s)], make([]T, size-cap(*s))...)
	}
}

func Del[S ~[]T, T any](s *S, f func(t *T) (del bool)) {
	for i := 0; i < len(*s); i++ {
		if f(&(*s)[i]) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			i -= 1
		}
	}
}

func DelPtr[S ~[]*T, T any](s *S, f func(t *T) (del bool)) {
	for i := 0; i < len(*s); i++ {
		if f((*s)[i]) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			i -= 1
		}
	}
}

func Range[T any](s []T) iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := 0; i < len(s); i++ {
			if !yield(i, &(s)[i]) {
				return
			}
		}
	}
}

func Search[T any](s []T, okf func(*T) bool) (k int, t *T) {
	for i := 0; i < len(s); i++ {
		if okf(&(s)[i]) {
			return i, &(s)[i]
		}
	}
	return -1, nil
}

// T是ptr时，使用AppendPtr
func Append[T any](s *[]T) *T {
	c, l := cap(*s), len(*s)
	if c > l {
		*s = (*s)[:l+1]
	} else {
		*s = append(*s, *new(T))
	}
	return &(*s)[l]
}

func AppendPtr[T any](s *[]*T) *T {
	c, l := cap(*s), len(*s)
	if c > l {
		*s = (*s)[:l+1]
		if (*s)[l] == nil {
			(*s)[l] = new(T)
		}
	} else {
		*s = append(*s, new(T))
	}
	return (*s)[l]
}
