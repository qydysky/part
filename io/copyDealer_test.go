package part

import (
	"bytes"
	"testing"
)

func Test_CopyDealer(t *testing.T) {
	sbuf := []byte(`123456`)

	mark := []byte(`23`)

	if !bytes.Contains(sbuf, mark) {
		t.Fatal()
	}
	tbuf1 := bytes.ReplaceAll(sbuf, mark, []byte(`11`))

	tbuf := bytes.NewBuffer([]byte{})

	if e := CopyDealer(tbuf, bytes.NewReader(sbuf), make([]byte, 3), func(data []byte) (dealed []byte, stop bool) {
		return bytes.ReplaceAll(data, mark, []byte(`11`)), false
	}); e != nil {
		t.Fatal(e)
	}

	if !bytes.Equal(tbuf.Bytes(), tbuf1) {
		t.Fatal()
	}
}
