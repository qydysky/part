package part

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	part "github.com/qydysky/part/io"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

func TestDir(t *testing.T) {
	Open("./test2").Delete()
	if Open("./test2").IsExist() {
		t.Fatal()
	}
	if f, e := DirFS("./test2").Open("1.txt"); e != nil {
		t.Fatal(e)
	} else if _, e := f.(*File).WriteRaw([]byte{'1'}, false); e != nil {
		t.Fatal(e)
	} else {
		f.(*File).Delete()
	}
	if !Open("./test2").IsExist() {
		t.Fatal()
	}
}

func TestMain(t *testing.T) {
	if runtime.GOOS == `windows` {
		if filepath.Join(`c:\`, "s/as") != `c:\s\as` {
			t.Fatal()
		}
	}
	if runtime.GOOS == `linux` {
		if filepath.Join(`/`, "s/as") != `/s/as` {
			t.Fatal()
		}
	}
}

func TestDirFs(t *testing.T) {
	f := New("./testdata", 0, true)
	if fs, err := f.DirFiles(); err != nil {
		t.Fatal(err)
	} else {
		if len(fs) != 1 {
			t.Fatal()
		}
		if fs[0] != "testdata"+string(os.PathSeparator)+"1.txt" {
			t.Fatal(fs[0])
		}
	}
}

func TestNewPath2(t *testing.T) {
	os.RemoveAll("./test")
	time.Sleep(time.Second)
	New("./test/test.log", 0, true).Create()
	New("./test/test2.log", 0, true).Create()
	time.Sleep(time.Second)
	os.RemoveAll("./test")
}

func TestNewPath(t *testing.T) {
	if runtime.GOOS == "linux" {
		f := New("/tmp/test/test.log", 0, true)
		f.Create()
		if !f.IsExist() {
			t.Fatal()
		}
		f.Delete()
	}
	if runtime.GOOS == "windows" {
		f := New("C:\\test\\test.log", 0, true)
		f.Create()
		if !f.IsExist() {
			t.Fatal()
		}
		f.Delete()
	}
	{
		f := New("./test/test.log", 0, true)
		f.Create()
		if !f.IsExist() {
			t.Fatal()
		}
		f.Delete()
	}
}

func TestWriteRawReadDelSync(t *testing.T) {
	f := New("rwd.txt", -6, true)
	if i, e := f.WriteRaw([]byte("sssa\n"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	var buf = make([]byte, 5)
	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		if !bytes.Equal(buf[:i], []byte("sssa\n")) {
			t.Fatal(i, string(buf), e)
		}
	}

	if i, e := f.Read(buf); i == 0 || e != nil {
		t.Fatal(i, e)
	} else {
		if !bytes.Equal(buf[:i], []byte("sssa\n")) {
			t.Fatal(i, string(buf), e)
		}
	}

	if e := f.Delete(); e != nil {
		t.Fatal(e)
	}
}

func TestWriteRawReadDel(t *testing.T) {
	f := New("rwd.txt", 0, false)
	f.Config.Coder = unicode.UTF8
	if i, e := f.WriteRaw([]byte("sssaaa"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.SeekIndex(0, AtOrigin); e != nil {
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

func TestSeek(t *testing.T) {
	f := New("rwd.txt", 0, false)
	if i, e := f.WriteRaw([]byte("12er4x3"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.SeekIndex(1, AtOrigin); e != nil {
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

	if e := f.SeekIndex(-1, AtEnd); e != nil {
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

func TestSeek2(t *testing.T) {
	f := New("rwd.txt", 0, false)
	if f.IsExist() {
		if e := f.Delete(); e != nil {
			t.Fatal(e)
		}
	}
	if i, e := f.WriteRaw([]byte("12345sser4x3"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	f.SeekIndex(0, AtOrigin)

	if e := f.SeekUntil([]byte("sser"), AtCurrent, 3, 1<<20); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(e)
	}

	if data, e := f.ReadAll(5, 1<<20); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("sser4x3")) {
		t.Fatal(string(data), data)
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
	if i, e := sf.WriteRaw([]byte("12er4x3"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	tf := New("t.txt", 0, true)
	if e := sf.CopyTo(tf, part.CopyConfig{BytePerSec: 1}, true); e != nil {
		t.Fatal(e)
	}

	sf.Delete()
	tf.Delete()
}

func TestReadUntil(t *testing.T) {
	f := New("s.txt", 0, false)
	if i, e := f.WriteRaw([]byte("18u3y7\ns99s9\nuqienbs\n"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if e := f.Sync(); e != nil {
		t.Fatal(e)
	}

	if e := f.SeekIndex(0, AtOrigin); e != nil {
		t.Fatal(e)
	}

	if data, e := f.ReadUntil([]byte{'\n'}, 5, 20); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("18u3y7")) {
		t.Fatal(string(data))
	}

	if data, e := f.ReadUntil([]byte{'\n'}, 5, 20); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("s99s9")) {
		t.Fatal(string(data))
	}

	if data, e := f.ReadUntil([]byte("s\n"), 5, 20); e != nil {
		t.Fatal(e)
	} else if !bytes.Equal(data, []byte("uqienb")) {
		t.Fatal(string(data))
	}

	if data, e := f.ReadUntil([]byte{'\n'}, 5, 20); e == nil || !errors.Is(e, io.EOF) || len(data) != 0 {
		t.Fatal(e)
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
	if i, e := sf.WriteRaw([]byte("测1试s啊是3大家看s法$和"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	tf := New("UTF8.txt", 0, true)
	tf.Config.Coder = unicode.UTF8
	if e := sf.CopyTo(tf, part.CopyConfig{BytePerSec: 5}, true); e != nil {
		t.Fatal(e)
	}

	if data, e := tf.ReadUntil([]byte{'\n'}, 3, 100); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(string(data), e)
	} else if !bytes.Equal(data, []byte("测1试s啊是3大家看s法$和")) {
		t.Fatal(string(data))
	}

	sf.Delete()
	tf.Delete()
}

func TestReadAll(t *testing.T) {
	sf := New("t.txt", 0, true)
	defer sf.Delete()

	if i, e := sf.WriteRaw([]byte("测1试s啊是3大家看s法$和"), true); i == 0 || e != nil {
		t.Fatal(e)
	}

	if data, e := sf.ReadAll(10, 1000); (e != nil && !errors.Is(e, io.EOF)) || !bytes.Equal(data, []byte("测1试s啊是3大家看s法$和")) {
		t.Fatal(e, string(data))
	}
}

func TestCreate(t *testing.T) {
	sf := New("t.txt", 0, true)
	defer sf.Delete()

	if sf.IsExist() {
		t.Fatal()
	}
	sf.Create()
	if !sf.IsExist() {
		t.Fatal()
	}
}

func TestIsRoot(t *testing.T) {
	sf := New("../t.txt", 0, true)
	sf.CheckRoot("testdata")
	if !strings.HasSuffix(sf.Delete().Error(), "path escapes from parent") {
		t.Fatal()
	}

	sf = New("testdata/1.txt", 0, true).CheckRoot("testdata")
	t.Log(sf.IsExist())
}
