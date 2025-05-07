package part

import (
	"errors"
	"runtime"
)

type BlocksI[T any] interface {
	// // eg
	//
	//	if tmpbuf, putBack, e := buf.Get(); e == nil {
	//		// do something with tmpbuf
	//		putBack()
	//	}
	Get() ([]T, func(), error)

	// // eg
	//
	//	if tmpbuf, e := buf.GetAuto(); e == nil {
	//		// do something with tmpbuf
	//	}
	GetAuto() ([]T, error)
}

type blocks[T any] struct {
	_    noCopy
	free chan int
	size int
	buf  []T
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

var ErrOverflow = errors.New("ErrOverflow")

func NewBlocks[T any](blockSize int, blockNum int) BlocksI[T] {
	p := &blocks[T]{
		size: blockSize,
		free: make(chan int, blockNum+1),
		buf:  make([]T, blockSize*blockNum),
	}
	for i := range blockNum {
		p.free <- i
	}
	return p
}

func (t *blocks[T]) Get() ([]T, func(), error) {
	t.gc()
	select {
	case offset := <-t.free:
		return t.buf[offset*t.size : (offset+1)*t.size], func() {
			clear(t.buf[offset*t.size : (offset+1)*t.size])
			t.free <- offset
		}, nil
	default:
		return nil, func() {}, ErrOverflow
	}
}

func (t *blocks[T]) GetAuto() (b []T, e error) {
	t.gc()
	select {
	case offset := <-t.free:
		b = t.buf[offset*t.size : (offset+1)*t.size]
		runtime.AddCleanup(&b, func(offset int) {
			clear(t.buf[offset*t.size : (offset+1)*t.size])
			t.free <- offset
		}, offset)
		return
	default:
		return nil, ErrOverflow
	}
}

func (t *blocks[T]) gc() {
	if len(t.free) == 0 {
		runtime.GC()
	}
}
