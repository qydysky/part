package part

import (
	"bytes"
	"io"
	"testing"
)

func Test_CopyIO(t *testing.T) {
	var s = make([]byte, 1<<17+2)
	s[1<<17-1] = '1'
	s[1<<17] = '2'
	s[1<<17+1] = '3'

	var w = &bytes.Buffer{}

	if e := Copy(bytes.NewReader(s), w, CopyConfig{1<<17 + 1, 1, 0, 0}); e != nil || w.Len() != 1<<17+1 || w.Bytes()[1<<17-1] != '1' || w.Bytes()[1<<17] != '2' {
		t.Fatal(e)
	}
}

func Test_rwc(t *testing.T) {
	rwc := RWC{R: func(p []byte) (n int, err error) { return 1, nil }}
	rwc.Close()
}

func Test_RW2Chan(t *testing.T) {
	{
		r, w := io.Pipe()
		_, rw := RW2Chan(nil, w)

		go func() {
			rw <- []byte{0x01}
		}()
		buf := make([]byte, 1<<16)
		n, _ := r.Read(buf)
		if buf[:n][0] != 1 {
			t.Error(`no`)
		}
	}

	{
		r, w := io.Pipe()
		rc, _ := RW2Chan(r, nil)

		go func() {
			w.Write([]byte{0x09})
		}()
		if b := <-rc; b[0] != 9 {
			t.Error(`no2`)
		}
	}

	{
		r, w := io.Pipe()
		rc, rw := RW2Chan(r, w)

		go func() {
			rw <- []byte{0x07}
		}()
		if b := <-rc; b[0] != 7 {
			t.Error(`no3`)
		}
	}
}
