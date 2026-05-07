package part

import (
	"io"
	"iter"
	"sync"

	pe "github.com/qydysky/part/errors/v2"
	pio "github.com/qydysky/part/io"
)

var (
	ActSlice = pe.Action[struct {
		ErrNoSameSliceIndex pe.Error
		ErrNoModified       pe.Error
		ErrSliceChange      pe.Error
	}](`ActSlice`)
)

type SliceIndexModified struct {
	p uintptr
	t uint64
}
type SliceIndex[T comparable] struct {
	l  sync.RWMutex
	in *SliceIndexNoLock[T]
}

func NewSliceIndex[T comparable](source []T) *SliceIndex[T] {
	return &SliceIndex[T]{
		in: NewSliceIndexNoLock(source),
	}
}
func (t *SliceIndex[T]) GetModified() SliceIndexModified {
	return t.in.GetModified()
}
func (t *SliceIndex[T]) HadModified(mt SliceIndexModified) (err error) {
	return t.in.HadModified(mt)
}
func (t *SliceIndex[T]) Merge(s, e int) {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.Merge(s, e)
}
func (t *SliceIndex[T]) Append(s, e int) {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.Append(s, e)
}
func (t *SliceIndex[T]) Size() (c int) {
	t.l.RLock()
	defer t.l.RUnlock()
	return t.in.Size()
}
func (t *SliceIndex[T]) Clear() {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.Clear()
}
func (t *SliceIndex[T]) Reset() {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.Clear()
}
func (t *SliceIndex[T]) Iter() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		t.l.RLock()
		defer t.l.RUnlock()
		index := 0
		for i := 0; i < len(t.in.buf); i += 2 {
			for z := t.in.buf[i]; z < t.in.buf[i+1]; z++ {
				if !yield(index, t.in.source[z]) {
					return
				}
				index += 1
			}
		}
	}
}
func (t *SliceIndex[T]) Equal(b []T) bool {
	matched := 0
	for _, v := range t.Iter() {
		if matched == len(b) {
			return false
		}
		if b[matched] != v {
			return false
		}
		matched += 1
	}
	return matched != len(b)-1
}
func (t *SliceIndex[T]) Read(p []T) (n int, err error) {
	t.l.Lock()
	defer t.l.Unlock()
	return t.in.Read(p)
}
func (t *SliceIndex[T]) IsEmpty() bool {
	t.l.RLock()
	defer t.l.RUnlock()
	return t.in.IsEmpty()
}
func (t *SliceIndex[T]) RemoveBack(n int) {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.RemoveBack(n)
}
func (t *SliceIndex[T]) RemoveFront(n int) {
	t.l.Lock()
	defer t.l.Unlock()
	t.in.RemoveFront(n)
}

type SliceIndexNoLock[T comparable] struct {
	buf []int
	// hash     [][16]byte
	source   []T
	modified SliceIndexModified
}

// 通过buf类似的操作方法，在现有buf上创建引用
func NewSliceIndexNoLock[T comparable](source []T) *SliceIndexNoLock[T] {
	return &SliceIndexNoLock[T]{
		buf:    []int{},
		source: source,
	}
}
func (t *SliceIndexNoLock[T]) GetModified() SliceIndexModified {
	return t.modified
}
func (t *SliceIndexNoLock[T]) HadModified(mt SliceIndexModified) (err error) {
	if t.modified.p != mt.p {
		return ActSlice.ErrNoSameSliceIndex
	} else if mt.t == t.modified.t {
		return ActSlice.ErrNoModified
	} else {
		return nil
	}
}

// 将source[s,e)合并到可读中
func (t *SliceIndexNoLock[T]) Merge(s, e int) {
	if len(t.buf) == 0 {
		t.buf = append(t.buf, s, e)
		t.modified.t += 1
		return
	} else {
		for i := 0; i < len(t.buf); i += 2 {
			if e < t.buf[i] {
				t.buf = append([]int{s, e}, t.buf...)
				t.modified.t += 1
				return
			} else if t.buf[i] <= e && e <= t.buf[i+1] {
				if s < t.buf[i] {
					t.buf[i] = s
					t.modified.t += 1
				}
				return
			} else if t.buf[i] <= s && s <= t.buf[i+1] {
				if t.buf[i+1] < e {
					t.buf[i+1] = e
					t.modified.t += 1
				}
				return
			}
		}
		t.buf = append(t.buf, s, e)
		t.modified.t += 1
		return
	}
}

// 将source[s,e)附加到可读后
func (t *SliceIndexNoLock[T]) Append(s, e int) {
	if i := len(t.buf) - 1; i >= 0 && s == t.buf[i] {
		if t.buf[i] < e {
			t.buf[i] = e
			// t.hash[(i-1)/2] = ph.Md5(t.source[t.buf[i-1]:t.buf[i]])
		}
	} else {
		t.buf = append(t.buf, s, e)
		// t.hash = append(t.hash, ph.Md5(t.source[s:e]))
	}
	t.modified.t += 1
}
func (t *SliceIndexNoLock[T]) FiliterAppend(s, e int, f func(*T) (pass bool)) {
	if i := len(t.buf) - 1; i >= 0 && s == t.buf[i] {
		if t.buf[i] < e {
			t.buf[i] = e
			t.modified.t += 1
		}
	} else {
		t.buf = append(t.buf, s, e)
		t.modified.t += 1
	}
}
func (t *SliceIndexNoLock[T]) Size() (c int) {
	for i := 0; i < len(t.buf); i += 2 {
		c += t.buf[i+1] - t.buf[i]
	}
	return
}
func (t *SliceIndexNoLock[T]) Clear() {
	t.buf = nil
	t.modified.t += 1
}
func (t *SliceIndexNoLock[T]) Reset() {
	t.buf = t.buf[:0]
	// t.hash = t.hash[:0]
	t.modified.t += 1
}
func (t *SliceIndexNoLock[T]) Iter() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		index := 0
		for i := 0; i < len(t.buf); i += 2 {
			for z := t.buf[i]; z < t.buf[i+1]; z++ {
				if !yield(index, t.source[z]) {
					return
				}
				index += 1
			}
		}
	}
}
func (t *SliceIndexNoLock[T]) Equal(b []T) bool {
	matched := 0
	for _, v := range t.Iter() {
		if matched == len(b) {
			return false
		}
		if b[matched] != v {
			return false
		}
		matched += 1
	}
	return matched != len(b)-1
}
func (t *SliceIndexNoLock[T]) Read(p []T) (n int, err error) {
	for len(p) > n && 2 <= len(t.buf) {
		// if ph.Md5(t.source[t.buf[0]:t.buf[1]]) != t.hash[0] {
		// 	panic("change")
		// }
		ln := copy(p[n:], t.source[t.buf[0]:t.buf[1]])
		if t.buf[1]-t.buf[0] == ln {
			t.buf = t.buf[2:]
			// t.hash = t.hash[1:]
		} else {
			t.buf[0] += ln
			// t.hash[0] = ph.Md5(t.source[t.buf[0]:t.buf[1]])
		}
		n += ln
	}
	t.modified.t += 1
	if len(t.buf) == 0 {
		err = io.EOF
	}
	return
}
func (t *SliceIndexNoLock[T]) IsEmpty() bool {
	return len(t.buf) == 0
}
func (t *SliceIndexNoLock[T]) RemoveBack(n int) {
	for i := len(t.buf) - 1; n > 0 && i >= 0; i-- {
		l := t.buf[i+1] - t.buf[i]
		if n < l {
			t.buf[i+1] = t.buf[i+1] - n
			t.modified.t += 1
		} else if n >= l {
			t.buf = t.buf[:len(t.buf)-2]
			n -= l
			t.modified.t += 1
		}
	}
}
func (t *SliceIndexNoLock[T]) RemoveFront(n int) {
	for i := 0; n > 0 && i < len(t.buf); i++ {
		l := t.buf[i+1] - t.buf[i]
		if n < l {
			t.buf[i] = t.buf[i] + n
			t.modified.t += 1
		} else if n >= l {
			t.buf = t.buf[2:]
			n -= l
			t.modified.t += 1
		}
	}
}

var _ io.WriterTo = pio.WrapIoWriteTo(&SliceIndexNoLock[byte]{})

func (t *SliceIndexNoLock[T]) WriteTo(w interface {
	Write(p []T) (n int, err error)
}) (n int64, err error) {
	for 2 <= len(t.buf) {
		ln, e := w.Write(t.source[t.buf[0]:t.buf[1]])
		n += int64(ln)
		if t.buf[1]-t.buf[0] == ln {
			t.buf = t.buf[2:]
		} else {
			t.buf[0] += ln
		}
		t.modified.t += 1
		if e != nil {
			err = e
			return
		}
	}
	return
}
