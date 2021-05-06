package part

import (
	"io"
)

//no close rc any time
//you can close wc, r, w.
func RW2Chan(r io.ReadCloser,w io.WriteCloser) (rc,wc chan[]byte) {
	if r != nil {
		rc = make(chan[]byte, 1<<16)
		go func(rc chan[]byte,r io.ReadCloser){
			for {
				buf := make([]byte, 1<<16)
				n,e := r.Read(buf)
				if n != 0 {
					rc <- buf[:n]
				} else if e != nil {
					close(rc)
					break
				}
			}
		}(rc,r)
	}
	
	if w != nil {
		wc = make(chan[]byte, 1<<16)
		go func(wc chan[]byte,w io.WriteCloser){
			for {
				buf :=<- wc
				if len(buf) == 0 {//chan close
					w.Close()
					break
				}
				_,e := w.Write(buf)
				if e != nil {
					close(wc)
					break
				}
			}
		}(wc,w)
	}
	return
}