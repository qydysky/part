package part

import (
	"io"

	pio "github.com/qydysky/part/io"
)

type RawReqRes struct {
	req  *pio.IOpipe
	res  *pio.IOpipe
	reqC chan struct{}
	resC chan struct{}
}

func NewRawReqRes() *RawReqRes {
	return &RawReqRes{req: pio.NewPipe(), res: pio.NewPipe(), reqC: make(chan struct{}), resC: make(chan struct{})}
}

func (t RawReqRes) ReqClose() error {
	select {
	case <-t.reqC:
		return nil
	default:
		close(t.reqC)
		return t.req.Close()
	}
}

func (t RawReqRes) ReqCloseWithError(e error) error {
	select {
	case <-t.reqC:
		return nil
	default:
		close(t.reqC)
		return t.req.CloseWithError(e)
	}
}

func (t RawReqRes) ResClose() error {
	select {
	case <-t.resC:
		return nil
	default:
		close(t.resC)
		return t.res.Close()
	}
}

func (t RawReqRes) ResCloseWithError(e error) error {
	select {
	case <-t.resC:
		return nil
	default:
		close(t.resC)
		return t.res.CloseWithError(e)
	}
}

func (t RawReqRes) Write(p []byte) (n int, err error) {
	select {
	case <-t.reqC:
		return t.res.Write(p)
	default:
		return 0, io.EOF
	}
}

func (t RawReqRes) Read(p []byte) (n int, err error) {
	select {
	case <-t.reqC:
		return 0, io.EOF
	default:
		return t.req.Read(p)
	}
}

func (t RawReqRes) ReqWrite(p []byte) (n int, err error) {
	select {
	case <-t.reqC:
		return 0, io.EOF
	default:
		return t.req.Write(p)
	}
}

func (t RawReqRes) ResRead(p []byte) (n int, err error) {
	select {
	case <-t.reqC:
		return t.res.Read(p)
	default:
		return 0, io.EOF
	}
}
