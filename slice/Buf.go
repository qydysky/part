package part

import (
	"runtime"
)

type BufI[T any] interface {
	// // eg
	//
	//	if tmpbuf, reachMax := buf.Get(); reachMax {
	//		// do something with tmpbuf
	//	}
	Get() []T
	Cache() int
}

type bufs[T any] struct {
	_    noCopy
	size uint64
	buf  chan []T
}

func NewBufs[T any](size uint64) BufI[T] {
	return &bufs[T]{
		size: size,
		buf:  make(chan []T, size),
	}
}

func (t *bufs[T]) Get() (tmpbuf []T) {
	if len(t.buf) > 0 {
		b := (<-t.buf)[:0]
		runtime.SetFinalizer(&b, func(objp any) {
			select {
			case t.buf <- *objp.(*[]T):
			default:
			}
		})
		return b
	} else {
		b := []T{}
		runtime.SetFinalizer(&b, func(objp any) {
			select {
			case t.buf <- *objp.(*[]T):
			default:
			}
		})
		return b
	}
}

func (t *bufs[T]) Cache() int {
	return len(t.buf)
}
