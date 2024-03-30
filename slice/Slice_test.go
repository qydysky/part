package part

import (
	"bytes"
	"testing"
	"unsafe"
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
	if e := b.RemoveFront(1); e == nil || e != ErrOverLen {
		t.Fatal()
	}
	if e := b.Append([]byte("abcdef")); e == nil || e != ErrOverMax {
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
	c.Append([]byte("abc"))
	if c.IsEmpty() || c.Size() != 3 {
		t.Fatal()
	}
}

func TestXxx3(t *testing.T) {
	var (
		c  = New[byte]()
		m  = New[byte]()
		s  = New[byte]()
		b1 = []byte("01234")
	)
	s.Append(b1)

	buf, ul := s.GetPureBufRLock()
	m.Append(buf)
	if e := m.AppendTo(c); e != nil {
		t.Fatal(e)
	}
	m.Reset()
	ul()

	buf1, ul1 := c.GetPureBufRLock()
	if !bytes.Equal(buf1, []byte("01234")) {
		t.Fatal()
	}
	ul1()
}

func TestXxx4(t *testing.T) {
	var (
		c  = New[byte]()
		b1 = []byte("01234333")
		b2 = []byte("01234")
	)
	c.Append(b1)
	b1[0] = 'a'
	c.SetFrom(b2)

	buf, ul := c.GetPureBufRLock()
	if !bytes.Equal(buf, []byte("01234")) {
		t.Fatal()
	}
	ul()
}

func TestXxx2(t *testing.T) {
	var c = New[byte]()
	c.Append([]byte("12345"))
	c.Append([]byte("67890"))
	first := c.GetCopyBuf()
	c.Reset()
	c.Append([]byte("abc"))
	c.Append([]byte("defg"))
	second := c.GetCopyBuf()
	c.Reset()
	c.Append([]byte("akjsdhfaksdjhf"))
	c.Append([]byte("9834719203857"))
	third := c.GetCopyBuf()
	c.Reset()
	if !bytes.Equal(first, []byte("1234567890")) {
		t.Fatal()
	}
	if !bytes.Equal(second, []byte("abcdefg")) {
		t.Fatal()
	}
	if !bytes.Equal(third, []byte("akjsdhfaksdjhf9834719203857")) {
		t.Fatal()
	}
}

func Test3(t *testing.T) {
	i := 1
	var s []*int
	var p = unsafe.Pointer(&s)
	s = append(s, &i, &i, &i)
	if unsafe.Pointer(&s) != p || cap(s) != 3 || len(s) != 3 {
		t.Fatal()
	}
	DelFront(&s, 3)
	if unsafe.Pointer(&s) != p || cap(s) != 3 || len(s) != 0 {
		t.Fatal()
	}
	AddFront(&s, &i)
	if unsafe.Pointer(&s) != p || cap(s) != 3 || len(s) != 1 {
		t.Fatal()
	}
	AddBack(&s, &i)
	if unsafe.Pointer(&s) != p || cap(s) != 3 || len(s) != 2 {
		t.Fatal()
	}
	DelBack(&s, 1)
	if unsafe.Pointer(&s) != p || cap(s) != 3 || len(s) != 1 {
		t.Fatal()
	}
}

func Test4(t *testing.T) {
	var c = New[byte]()
	var w = make(chan struct{})

	c.Append([]byte("12345"))

	buf, unlock := c.GetPureBufRLock()

	go func() {
		w <- struct{}{}
		c.Reset()
		t.Log(c.Append([]byte("22345")))
		t.Log(c.buf)
		w <- struct{}{}
	}()

	<-w

	if !bytes.Equal(buf, []byte("12345")) {
		t.Fatal()
	}

	unlock()

	<-w

	if !bytes.Equal(buf, []byte("22345")) {
		t.Fatal(buf)
	}
}
