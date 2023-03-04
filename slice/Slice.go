package part

import (
	"errors"
	"sync"
	"unsafe"
)

type Buf[T any] struct {
	maxsize  int
	bufsize  int
	modified Modified
	buf      []T
	sync.RWMutex
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
	t.Lock()
	defer t.Unlock()
	t.buf = nil
	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) Size() int {
	t.RLock()
	defer t.RUnlock()

	return t.bufsize
}

func (t *Buf[T]) IsEmpty() bool {
	t.RLock()
	defer t.RUnlock()

	return t.bufsize == 0
}

func (t *Buf[T]) Reset() {
	t.Lock()
	defer t.Unlock()

	t.bufsize = 0
	t.modified.t += 1
}

func (t *Buf[T]) Append(data []T) error {
	t.Lock()
	defer t.Unlock()

	if t.maxsize != 0 && len(t.buf)+len(data) > t.maxsize {
		return errors.New("超出设定maxsize")
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

func (t *Buf[T]) RemoveFront(n int) error {
	if n <= 0 {
		return nil
	}

	t.Lock()
	defer t.Unlock()

	if t.bufsize < n {
		return errors.New("尝试移除的数值大于长度")
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

	t.Lock()
	defer t.Unlock()

	if t.bufsize < n {
		return errors.New("尝试移除的数值大于长度")
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
	t.Lock()
	defer t.Unlock()

	t.modified.t += 1
}

func (t *Buf[T]) GetModified() Modified {
	t.RLock()
	defer t.RUnlock()

	return t.modified
}

func (t *Buf[T]) HadModified(mt Modified) (modified bool, err error) {
	t.RLock()
	defer t.RUnlock()

	if t.modified.p != mt.p {
		err = errors.New("不能对比不同buf")
	}
	modified = t.modified.t != mt.t
	return
}

// unsafe
func (t *Buf[T]) GetPureBuf() (buf []T) {
	t.RLock()
	defer t.RUnlock()

	return t.buf[:t.bufsize]
}

func (t *Buf[T]) GetCopyBuf() (buf []T) {
	t.RLock()
	defer t.RUnlock()

	buf = make([]T, t.bufsize)
	copy(buf, t.buf[:t.bufsize])
	return
}

// Need to lock when processing buf or Make sure buf only processing by GetBufCopy
func (t *Buf[T]) GetBufCopy(buf *[]T) {
	t.RLock()
	defer t.RUnlock()

	if len(*buf) == 0 {
		*buf = make([]T, t.bufsize)
	} else if len(*buf) < t.bufsize {
		*buf = append(*buf, make([]T, t.bufsize-len(*buf))...)
	} else {
		*buf = (*buf)[:t.bufsize]
	}
	copy(*buf, t.buf[:t.bufsize])
}
