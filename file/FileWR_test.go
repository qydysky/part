package part

import (
	"bytes"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

func TestWriteReadDelSync(t *testing.T) {
	f := New("rwd.txt", 0, true)
	if i, e := f.Write([]byte("sss"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	var buf = make([]byte, 3)
	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		for _, v := range buf {
			if v != 's' {
				t.Fatal(v)
			}
		}
	}

	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		for _, v := range buf {
			if v != 's' {
				t.Fatal(v)
			}
		}
	}

	if e := f.Delete(); e != nil {
		t.Fatal(e)
	}
}

func TestWriteReadDel(t *testing.T) {
	f := New("rwd.txt", 0, false)
	f.Config.Coder = unicode.UTF8
	if i, e := f.Write([]byte("sssaaa"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.Seed(0); e != nil {
		t.Fatal(e)
	}

	var buf = make([]byte, 3)
	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else if !bytes.Equal(buf, []byte("sss")) {
		t.Fatal(string(buf))
	}

	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else if !bytes.Equal(buf, []byte("aaa")) {
		t.Fatal(string(buf))
	}

	if e := f.Close(); e != nil {
		t.Fatal(e)
	}

	if e := f.Delete(); e != nil {
		t.Fatal(e)
	}
}

func TestSeed(t *testing.T) {
	f := New("rwd.txt", 0, false)
	if i, e := f.Write([]byte("12er4x3"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.Seed(1); e != nil {
		t.Fatal(e)
	}

	var buf = make([]byte, 1)
	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		if buf[0] != '2' {
			t.Fatal(buf[0])
		}
	}

	if e := f.Seed(-1); e != nil {
		t.Fatal(e)
	}

	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		if buf[0] != '3' {
			t.Fatal(buf[0])
		}
	}

	if e := f.Close(); e != nil {
		t.Fatal(e)
	}

	if e := f.Delete(); e != nil {
		t.Fatal(e)
	}
}

func TestCopy(t *testing.T) {
	sf := New("s.txt", 0, true)
	if i, e := sf.Write([]byte("12er4x3"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	tf := New("t.txt", 0, true)
	if e := sf.CopyTo(tf, 1, true); e != nil {
		t.Fatal(e)
	}

	sf.Delete()
	tf.Delete()
}

func TestReadUntil(t *testing.T) {
	f := New("s.txt", 0, false)
	if i, e := f.Write([]byte("18u3y7\ns99s9\n"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.Sync(); e != nil {
		t.Fatal(e)
	}

	if e := f.Seed(0); e != nil {
		t.Fatal(e)
	}

	if data, e := f.ReadUntil('\n', 5, 20); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("18u3y7")) {
		t.Fatal(string(data))
	}

	t.Log(f.Config.CurIndex)

	if data, e := f.ReadUntil('\n', 5, 20); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("s99s9")) {
		t.Fatal(string(data))
	}

	if e := f.Close(); e != nil {
		t.Fatal(e)
	}

	if e := f.Delete(); e != nil {
		t.Fatal(e)
	}
}

func TestEncoderDecoder(t *testing.T) {
	sf := New("GBK.txt", 0, true)
	sf.Config.Coder = simplifiedchinese.GBK
	if i, e := sf.Write([]byte("测1试s啊是3大家看s法$和"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	tf := New("UTF8.txt", 0, true)
	tf.Config.Coder = unicode.UTF8
	if e := sf.CopyTo(tf, 5, true); e != nil {
		t.Fatal(e)
	}

	if data, e := tf.ReadUntil('\n', 3, 100); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("测1试s啊是3大家看s法$和")) {
		t.Fatal(string(data))
	}

	sf.Delete()
	tf.Delete()
}