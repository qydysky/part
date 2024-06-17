package part

import (
	"sync"
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

func (t *Buf[T]) Append(data []T) error {
	t.l.Lock()
	defer t.l.Unlock()

	if t.maxsize != 0 && len(t.buf)+len(data) > t.maxsize {
		return ErrOverMax
	} else if len(t.buf) == 0 {
		t.buf = make([]T, len(data))
	} else {
		diff := len(t.buf) - t.bufsize - len(data)
		if diff < 0 {
			t.buf = append(t.buf, make([]T, -diff)...)
		}
	}
	t.bufsize += copy(t.buf[t.bufsize:], data)
	t.modified.t += 1
	return nil
}

func (t *Buf[T]) SetFrom(data []T) error {
	t.l.Lock()
	defer t.l.Unlock()

	if t.maxsize != 0 && len(t.buf)+len(data) > t.maxsize {
		return ErrOverMax
	} else if len(t.buf) == 0 {
		t.buf = make([]T, len(data))
	} else {
		diff := len(t.buf) - t.bufsize - len(data)
		if diff < 0 {
			t.buf = append(t.buf, make([]T, -diff)...)
		}
	}
	t.bufsize = copy(t.buf, data)
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

func DelFront[S ~[]T, T any](s *S, beforeIndex int) {
	*s = (*s)[:copy(*s, (*s)[beforeIndex:])]
}

func AddFront[S ~[]*T, T any](s *S, t *T) {
	*s = append(*s, nil)
	*s = (*s)[:1+copy((*s)[1:], *s)]
	(*s)[0] = t
}

func DelBack[S ~[]T, T any](s *S, fromIndex int) {
	*s = (*s)[:fromIndex]
}

func AddBack[S ~[]*T, T any](s *S, t *T) {
	*s = append(*s, t)
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
			i-=1
		}
	}
}
