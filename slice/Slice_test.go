package part

import (
	"bytes"
	"testing"
)

func TestXxx(t *testing.T) {
	var (
		b = New[byte](5)
		c = New[byte]()
	)
	if !b.IsEmpty() {
		t.Fatal()
	}
	b.Append([]byte("abc"))
	if b.IsEmpty() || b.Size() != 3 {
		t.Fatal()
	}
	if _, e := b.HadModified(c.GetModified()); e == nil {
		t.Fatal()
	}
	bc := b.GetCopyBuf()
	bt := b.GetModified()
	b.RemoveFront(1)
	if b.IsEmpty() || b.Size() != 2 || !bytes.Equal(bc, []byte("abc")) ||
		!bytes.Equal(b.GetPureBuf(), []byte("bc")) || bt == b.GetModified() {
		t.Fatal()
	}
	b.RemoveBack(1)
	if b.IsEmpty() || b.Size() != 1 || !bytes.Equal(bc, []byte("abc")) || !bytes.Equal(b.GetPureBuf(), []byte("b")) {
		t.Fatal()
	}
	b.Reset()
	if !b.IsEmpty() || b.Size() != 0 || !bytes.Equal(bc, []byte("abc")) || !bytes.Equal(b.GetPureBuf(), []byte("")) {
		t.Fatal()
	}
	if e := b.RemoveFront(1); e == nil || e.Error() != "尝试移除的数值大于长度" {
		t.Fatal()
	}
	if e := b.Append([]byte("abcdef")); e == nil || e.Error() != "超出设定maxsize" {
		t.Fatal()
	}
	b.Clear()
	if !b.IsEmpty() || b.Size() != 0 {
		t.Fatal()
	}
	b.Append([]byte("abc"))
	if b.IsEmpty() || b.Size() != 3 {
		t.Fatal()
	}
}
