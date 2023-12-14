package part

import (
	"errors"
)

type blocks[T any] struct {
	free chan int
	size int
	buf  []T
}

var ErrOverflow = errors.New("ErrOverflow")

func NewBlocks[T any](blockSize int, blockNum int) *blocks[T] {
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

// // eg
//
//	if tmpbuf, putBack, e := buf.Get(); e == nil {
//		clear(tmpbuf)
//		// do something with tmpbuf
//		putBack()
//	}
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
