package part

import (
	"io"
	"sync/atomic"

	pio "github.com/qydysky/part/io"
)

type RawReqRes struct {
	req  *pio.IOpipe
	res  *pio.IOpipe
	reqC *atomic.Bool
	resC *atomic.Bool
}

func NewRawReqRes() *RawReqRes {
	p := &RawReqRes{req: pio.NewPipe(), res: pio.NewPipe(), reqC: &atomic.Bool{}, resC: &atomic.Bool{}}
	return p
}

func (t RawReqRes) ReqClose() error {
	if !t.reqC.Swap(true) {
		return t.req.Close()
	}
	return nil
}

func (t RawReqRes) ReqCloseWithError(e error) error {
	if !t.reqC.Swap(true) {
		return t.req.CloseWithError(e)
	}
	return nil
}

func (t RawReqRes) ResClose() error {
	if !t.resC.Swap(true) {
		return t.res.Close()
	}
	return nil
}

func (t RawReqRes) ResCloseWithError(e error) error {
	if !t.resC.Swap(true) {
		return t.res.CloseWithError(e)
	}
	return nil
}

func (t RawReqRes) Write(p []byte) (n int, err error) {
	if t.reqC.Load() {
		return t.res.Write(p)
	}
	return 0, io.EOF
}

func (t RawReqRes) Read(p []byte) (n int, err error) {
	if !t.reqC.Load() {
		return t.req.Read(p)
	}
	return 0, io.EOF
}

func (t RawReqRes) ReqWrite(p []byte) (n int, err error) {
	if !t.reqC.Load() {
		return t.req.Write(p)
	}
	return 0, io.EOF
}

func (t RawReqRes) ResRead(p []byte) (n int, err error) {
	if t.reqC.Load() {
		return t.res.Read(p)
	}
	return 0, io.EOF
}
