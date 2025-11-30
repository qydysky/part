package part

import (
	"errors"
	"runtime"
	"sync"
)

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type BlocksI[T any] interface {
	// // eg
	//
	//	if tmpbuf, putBack, e := buf.Get(); e == nil {
	// 		tmpbuf = append(tmpbuf[:0], b...)
	//		// do something with tmpbuf
	//		putBack()
	//	}
	Get() ([]T, func(), error)

	// // eg
	//
	//	if tmpbuf, e := buf.GetAuto(); e == nil {
	// 		tmpbuf = append(tmpbuf[:0], b...)
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
	select {
	case offset := <-t.free:
		return t.buf[offset*t.size : (offset+1)*t.size], func() {
			t.free <- offset
		}, nil
	default:
		return nil, func() {}, ErrOverflow
	}
}

func (t *blocks[T]) GetAuto() (b []T, e error) {
	select {
	case offset := <-t.free:
		b = t.buf[offset*t.size : (offset+1)*t.size]
		runtime.AddCleanup(&b, func(offset int) {
			t.free <- offset
		}, offset)
		return
	default:
		return nil, ErrOverflow
	}
}

type FlexBlocksI[T any] interface {
	// // eg
	//
	//	if tmpbuf, putBack, e := buf.Get(); e == nil {
	// 		tmpbuf = append(tmpbuf[:0], b...)
	//		// do something with tmpbuf
	//		putBack(tmpbuf)
	//	}
	Get() ([]T, func([]T), error)
}

type flexBlocks[T any] struct {
	_    noCopy
	free chan int
	buf  [][]T
}

func NewFlexBlocks[T any](blockNum int) FlexBlocksI[T] {
	p := &flexBlocks[T]{
		free: make(chan int, blockNum+1),
		buf:  make([][]T, blockNum),
	}
	for i := range blockNum {
		p.free <- i
	}
	return p
}

func (t *flexBlocks[T]) Get() ([]T, func([]T), error) {
	select {
	case offset := <-t.free:
		return t.buf[offset], func(ts []T) {
			t.buf[offset] = ts
			t.free <- offset
		}, nil
	default:
		return nil, func(_ []T) {}, ErrOverflow
	}
}

type PoolBlocksI[T any] interface {
	Get() *[]T
	Put(*[]T)
}

type poolBlocks[T any] struct {
	pool sync.Pool
}

func NewPoolBlocks[T any](initf ...func() *[]T) PoolBlocksI[T] {
	t := &poolBlocks[T]{}
	t.pool.New = func() any {
		if len(initf) > 0 {
			return initf[0]()
		} else {
			return new([]T)
		}
	}
	return t
}

func (t *poolBlocks[T]) Get() (i *[]T) {
	return t.pool.Get().(*[]T)
}

func (t *poolBlocks[T]) Put(i *[]T) {
	t.pool.Put(i)
}

type PoolBlockI[T any] interface {
	Get() *T
	Put(*T)
}

type poolBlock[T any] struct {
	pool sync.Pool
}

func NewPoolBlock[T any](initf ...func() *T) PoolBlockI[T] {
	t := &poolBlock[T]{}
	t.pool.New = func() any {
		if len(initf) > 0 {
			return initf[0]()
		} else {
			return new(T)
		}
	}
	return t
}

func (t *poolBlock[T]) Get() (i *T) {
	return t.pool.Get().(*T)
}

func (t *poolBlock[T]) Put(p *T) {
	t.pool.Put(p)
}
