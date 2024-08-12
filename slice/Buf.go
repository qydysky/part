package part

import (
	"runtime"
	"sync/atomic"
)

type BufI[T any] interface {
	// // eg
	//
	//	if tmpbuf, e := buf.Get(); e == nil {
	//		// do something with tmpbuf
	//	}
	Get() []T
	CacheCount() int64
}

type bufs[T any] struct {
	_   noCopy
	num atomic.Int64
	buf [][]T
}

func NewBufs[T any]() BufI[T] {
	return &bufs[T]{
		buf: [][]T{},
	}
}

func (t *bufs[T]) Get() (b []T) {
	if len(t.buf) > 0 {
		b = t.buf[0][:0]
		t.buf = t.buf[:copy(t.buf, t.buf[1:])]
		t.num.Add(-1)
		return
	} else {
		b = []T{}
	}
	runtime.SetFinalizer(&b, func(objp any) {
		t.buf = append(t.buf, *objp.(*[]T))
		t.num.Add(1)
	})
	return
}

func (t *bufs[T]) CacheCount() int64 {
	return t.num.Load()
}
