package part

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
	"unsafe"
)

func Test6(t *testing.T) {
	var b = strings.NewReader("1234567890")
	buf := New[byte]()
	if n, err := AsioReaderBuf(buf, b); err != nil {
		t.Fatal(err)
	} else if string(buf.getPureBuf()) != "1234567890" || n != 10 {
		t.Fatal(n)
	}
}

func Test7(t *testing.T) {
	var b = strings.NewReader("1234567890")
	buf := New[byte](8)
	n, err := AsioReaderBuf(buf, b)
	if err != nil || n != 8 {
		t.Fatal(err)
	}
	_, err = AsioReaderBuf(buf, b)
	if err != nil && err != ErrOverMax {
		t.Fatal(err)
	}
}

func Benchmark4(b *testing.B) {
	var data = strings.NewReader("1234567890")
	buf := New[byte]()
	for b.Loop() {
		if _, err := AsioReaderBuf(buf, data); err != nil {
			b.Fatal(err)
		} else if string(buf.getPureBuf()) != "1234567890" {
			b.Fatal()
		} else {
			buf.Reset()
			data.Seek(0, io.SeekStart)
		}
	}
}

func Benchmark5(b *testing.B) {
	var data = strings.NewReader("1234567890")
	buf := make([]byte, 4000)
	for b.Loop() {
		if n, err := data.Read(buf); err != nil {
			b.Fatal(err)
		} else if string(buf[:n]) != "1234567890" {
			b.Fatal()
		} else {
			data.Seek(0, io.SeekStart)
		}
	}
}

func TestDel(t *testing.T) {
	var s = []int{1, 2, 3, 4, 4, 6, 4, 7}
	Del(&s, func(t *int) (del bool) {
		return *t == 4
	})
	t.Log(s)
	if s[3] != 6 || s[4] != 7 {
		t.FailNow()
	}
}

func TestResize(t *testing.T) {
	var s = make([]byte, 10)
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	s = s[:0]
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	Resize(&s, 8)
	if len(s) != 8 {
		t.FailNow()
	}
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	Resize(&s, 4)
	if len(s) != 4 {
		t.FailNow()
	}
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	Resize(&s, 11)
	if len(s) != 11 {
		t.FailNow()
	}
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	Resize(&s, 25)
	if len(s) != 25 {
		t.FailNow()
	}
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
	Resize(&s, 3)
	if len(s) != 3 {
		t.FailNow()
	}
	t.Log(unsafe.Pointer(&s), len(s), cap(s))
}

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
	var s []int
	var p = unsafe.Pointer(&s)
	s = append(s, i, i, i)
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

func Test(t *testing.T) {
	var buf = []int{1, 2, 3}
	var v int = 1
	LoopAddBack(&buf, &v)
	if !slices.Equal(buf, []int{2, 3, 1}) {
		t.Fatal()
	}
	LoopAddFront(&buf, &v)
	if !slices.Equal(buf, []int{1, 2, 3}) {
		t.Fatal()
	}
}

func Benchmark1(b *testing.B) {
	var buf = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	var t int = 1
	for b.Loop() {
		LoopAddBack(&buf, &t)
	}
}

func Benchmark3(b *testing.B) {
	var buf = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	var t int = 1
	for b.Loop() {
		LoopAddFront(&buf, &t)
	}
}

func Benchmark(b *testing.B) {
	// var buf []ie = []ie{{make([]byte, 50002)}, {make([]byte, 50001)}, {make([]byte, 50000)}}
	var buf1 = make([]byte, 50002)
	var f = func(a []byte) byte {
		a[0] = '1'
		return a[0]
	}
	for b.Loop() {
		_ = f(buf1)
		// slices.SortFunc(buf, func(a, b ie) int { return len(a.key) - len(b.key) })
	}
}

type ie struct {
	key []byte
}

func Benchmark2(b *testing.B) {
	data := make([]byte, 50000)
	var buf = []ie{{data}}
	for b.Loop() {
		buf = buf[:0]
		t := Append(&buf)
		t.key = append(t.key[:0], data...)
	}
}

func Test2(t *testing.T) {
	b2 := testing.Benchmark(Benchmark2)
	if a := b2.AllocedBytesPerOp(); a > 0 {
		t.Fatal(a)
	}
}

func Test5(t *testing.T) {
	type L int
	var p []*L

	for i := 0; i < 10; i++ {
		if AppendPtr(&p) == nil {
			t.Fatal()
		}
	}
}
