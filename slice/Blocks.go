package part

import (
	"errors"
)

type BlocksI[T any] interface {
	// // eg
	//
	//	if tmpbuf, putBack, e := buf.Get(); e == nil {
	//		clear(tmpbuf)
	//		// do something with tmpbuf
	//		putBack()
	//	}
	Get() ([]T, func(), error)
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
	for i := 0; i < blockNum; i++ {
		p.free <- i
	}
	return p
}

func (t *blocks[T]) Get() ([]T, func(), error) {
	select {
	case offset := <-t.free:
		offset *= t.size
		return t.buf[offset : offset+t.size], func() {
			clear(t.buf[offset : offset+t.size])
			t.free <- offset
		}, nil
	default:
		return nil, func() {}, ErrOverflow
	}
}
