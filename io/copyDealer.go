package part

import (
	"errors"
	"io"
)

var ErrCopyDealerStop = errors.New(`ErrCopyDealerStop`)
var ErrCopyDealerBufOF = errors.New(`ErrCopyDealerBufOF`)

// buf 最小需要匹配尺寸*1.5 才能有效
func CopyDealer(w io.Writer, r io.Reader, buf []byte, dealers ...func(data []byte) (dealed []byte, stop bool)) (e error) {
	n := 0
	left := 0
	si := len(buf) / 3
	for {
		n, e = r.Read(buf[left : si*2])
		if n > 0 {
			n += left
			for i := 0; i < len(dealers); i++ {
				if dealed, stop := dealers[i](buf[:n]); stop {
					return ErrCopyDealerStop
				} else if len(dealed) > len(buf) {
					return ErrCopyDealerBufOF
				} else {
					n = copy(buf, dealed)
				}
			}
		} else if e != nil {
			if errors.Is(e, io.EOF) {
				e = nil
				if left > 0 {
					if _, e := w.Write(buf[:left]); e != nil {
						if errors.Is(e, io.EOF) {
							e = nil
						}
						return e
					}
				}
			}
			return e
		}
		if wn, e := w.Write(buf[:min(si, n)]); wn > 0 {
			left = copy(buf, buf[wn:n])
		} else if e != nil {
			if errors.Is(e, io.EOF) {
				e = nil
			}
			return e
		}
	}
}
