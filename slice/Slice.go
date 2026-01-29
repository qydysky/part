package part

import (
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
	t.l.Lock()
	defer t.l.Unlock()
	t.buf = nil
	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) Size() int {
	t.l.RLock()
	defer t.l.RUnlock()

	return t.bufsize
}

func (t *Buf[T]) IsEmpty() bool {
	t.l.RLock()
	defer t.l.RUnlock()

	return t.bufsize == 0
}

func (t *Buf[T]) Reset() {
	t.l.Lock()
	defer t.l.Unlock()

	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) AppendTo(to *Buf[T]) error {
	buf, unlock := t.GetPureBufRLock()
	defer unlock()
	return to.Append(buf)
}

var ErrOverMax = perrors.New("slices.Append", "ErrOverMax")

func (t *Buf[T]) Cap() int {
	t.l.RLock()
	defer t.l.RUnlock()
	return cap(t.buf)
}

// actual buf size may larger then maxsize
func AsioReaderBuf(t *Buf[byte], r io.Reader) (n int, err error) {
	t.l.Lock()
	defer t.l.Unlock()

	if cap(t.buf) == 0 {
		if t.maxsize > 0 {
			t.buf = append(t.buf[:cap(t.buf)], make([]byte, min(4096, t.maxsize))...)
		} else {
			t.buf = append(t.buf[:cap(t.buf)], make([]byte, min(4096))...)
		}
	} else if leftS := cap(t.buf) - t.bufsize; leftS == 0 {
		if t.maxsize > 0 && t.maxsize < 2*cap(t.buf) {
			return 0, ErrOverMax
		}
		t.buf = append(t.buf[:cap(t.buf)], make([]byte, cap(t.buf))...)
	}

	t.buf = t.buf[:cap(t.buf)]
	n, err = r.Read(t.buf[t.bufsize:])
	if n > 0 {
		t.bufsize += n
		t.modified.t += 1
	}
	return
}

func (t *Buf[T]) ExpandCapTo(size int) {
	t.l.Lock()
	defer t.l.Unlock()
	if cap(t.buf) >= size {
		return
	} else {
		t.buf = append(t.buf[:cap(t.buf)], make([]T, size-cap(t.buf))...)
	}
}

func (t *Buf[T]) Append(data []T) error {
	t.l.Lock()
	defer t.l.Unlock()
	return t.append(data)
}

func (t *Buf[T]) append(data []T) error {
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
	t.e = t.bufp.append(data)
	return t
}

func (t *Buf[T]) Appends(f func(ba *BufAppendsI[T])) error {
	t.l.Lock()
	defer t.l.Unlock()
	ba := &BufAppendsI[T]{bufp: t}
	f(ba)
	return ba.e
}

func (t *Buf[T]) SetFrom(data []T) error {
	t.l.Lock()
	defer t.l.Unlock()

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

	t.l.Lock()
	defer t.l.Unlock()
	return t.removeFront(n)
}

func (t *Buf[T]) removeFront(n int) error {
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

	t.l.Lock()
	defer t.l.Unlock()
	return t.removeBack(n)
}

func (t *Buf[T]) removeBack(n int) error {
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
	t.l.Lock()
	defer t.l.Unlock()

	t.modified.t += 1
}

func (t *Buf[T]) GetModified() Modified {
	t.l.RLock()
	defer t.l.RUnlock()

	return t.modified
}

var ErrNoSameBuf = perrors.New("slices.HadModified", "ErrNoSameBuf")

func (t *Buf[T]) HadModified(mt Modified) (modified bool, err error) {
	t.l.RLock()
	defer t.l.RUnlock()

	if t.modified.p != mt.p {
		err = ErrNoSameBuf
	}
	modified = t.modified.t != mt.t
	return
}

// unsafe
func (t *Buf[T]) GetPureBuf() (buf []T) {
	t.l.RLock()
	defer t.l.RUnlock()

	return t.getPureBuf()
}
func (t *Buf[T]) getPureBuf() (buf []T) {
	return t.buf[:t.bufsize]
}

// must call unlock
//
// buf will no modify before unlock
//
// modify func(eg Reset) with block until unlock
//
// unsafe
func (t *Buf[T]) GetPureBufRLock() (buf []T, unlock func()) {
	t.l.RLock()
	return t.buf[:t.bufsize], t.l.RUnlock
}

func (t *Buf[T]) GetCopyBuf() (buf []T) {
	t.l.RLock()
	defer t.l.RUnlock()

	buf = make([]T, t.bufsize)
	copy(buf, t.buf[:t.bufsize])
	return
}

type BufLockM[T any] struct {
	*Buf[T]
}

func (b *BufLockM[T]) GetPureBuf() (buf []T) {
	return b.getPureBuf()
}

func (b *BufLockM[T]) RemoveBack(n int) error {
	return b.removeBack(n)
}

func (b *BufLockM[T]) RemoveFront(n int) error {
	return b.removeFront(n)
}

func (b *BufLockM[T]) Unlock() {
	b.l.Unlock()
}

func (b *BufLockM[T]) Append(data []T) error {
	return b.append(data)
}

type BufLockMI[T any] interface {
	Unlock()
	GetPureBuf() []T
	RemoveFront(int) error
	RemoveBack(int) error
	Append(data []T) error
}

func (t *Buf[T]) GetLock() (i BufLockMI[T]) {
	t.l.Lock()
	i = &BufLockM[T]{t}
	return i
}

func (t *Buf[T]) Read(b []T) (n int, e error) {
	for t.Size() == 0 {
		time.Sleep(time.Millisecond * 100)
	}
	method := t.GetLock()
	n = copy(b, method.GetPureBuf())
	e = method.RemoveFront(n)
	method.Unlock()
	return
}

func (t *Buf[T]) Write(b []T) (n int, e error) {
	return len(b), t.Append(b)
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
