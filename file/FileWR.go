package part

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	l "github.com/qydysky/part/limit"
	encoder "golang.org/x/text/encoding"
)

var (
	ErrFilePathTooLong  = errors.New("ErrFilePathTooLong")
	ErrNewFileCantSeed  = errors.New("ErrNewFileCantSeed")
	ErrFailToLock       = errors.New("ErrFailToLock")
	ErrMaxReadSizeReach = errors.New("ErrMaxReadSizeReach")
)

type File struct {
	Config Config
	file   *os.File
	wr     io.Writer
	rr     io.Reader
	cu     int64
	l      sync.RWMutex
}

type Config struct {
	FilePath  string //文件路径
	CurIndex  int64  //初始化光标位置
	AutoClose bool   //自动关闭句柄

	// wrap with encoder
	//https://pkg.go.dev/golang.org/x/text/encoding#section-directories
	Coder encoder.Encoding
}

func New(filePath string, curIndex int64, autoClose bool) *File {
	return &File{
		Config: Config{
			FilePath:  filePath,
			CurIndex:  curIndex,
			AutoClose: autoClose,
		},
	}
}

func (t *File) CopyTo(to *File, byteInSec int64, tryLock bool) error {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	to.getRWCloser()
	if t.Config.AutoClose {
		defer to.Close()
	}

	if tryLock {
		if !to.l.TryLock() {
			return ErrFailToLock
		}
	} else {
		to.l.Lock()
	}
	defer to.l.Unlock()

	return transferIO(t.read(), to.write(), byteInSec)
}

func (t *File) CopyToIoWriter(to io.Writer, byteInSec int64, tryLock bool) error {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	return transferIO(t.read(), to, byteInSec)
}

func (t *File) Write(data []byte, tryLock bool) (int, error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if tryLock {
		if !t.l.TryLock() {
			return 0, ErrFailToLock
		}
	} else {
		t.l.Lock()
	}
	defer t.l.Unlock()

	return t.write().Write(data)
}

func (t *File) Read(data []byte) (int, error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return 0, ErrFailToLock
	}
	defer t.l.RUnlock()

	return t.read().Read(data)
}

func (t *File) ReadUntil(separation byte, perReadSize int, maxReadSize int) (data []byte, e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return nil, ErrFailToLock
	}
	defer t.l.RUnlock()

	var (
		tmpArea = make([]byte, perReadSize)
		n       int
		reader  = t.read()
	)

	for maxReadSize > 0 {
		n, e = reader.Read(tmpArea)

		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:n], []byte{separation}); i != -1 {
			if n-i-1 != 0 {
				t.file.Seek(-int64(n-i-1), 1)
			}
			if i != 0 {
				data = append(data, tmpArea[:i]...)
			}
			break
		} else {
			data = append(data, tmpArea[:n]...)
		}
	}

	if maxReadSize <= 0 {
		e = ErrMaxReadSizeReach
	}

	return
}

func (t *File) ReadAll(perReadSize int, maxReadSize int) (data []byte, e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return nil, ErrFailToLock
	}
	defer t.l.RUnlock()

	var (
		tmpArea = make([]byte, perReadSize)
		n       = 0
		reader  = t.read()
	)

	for maxReadSize > 0 {
		n, e = reader.Read(tmpArea)

		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		data = append(data, tmpArea[:n]...)
	}

	if maxReadSize <= 0 {
		e = ErrMaxReadSizeReach
	}

	return
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted according to whence: 0 means relative to the origin of the file, 1 means relative to the current offset, and 2 means relative to the end. It returns the new offset and an error, if any. The behavior of Seek on a file opened with O_APPEND is not specified.
func (t *File) Seed(index int64, whence int) (e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryLock() {
		return ErrFailToLock
	}
	defer t.l.Unlock()

	t.cu, e = t.file.Seek(index, whence)

	return nil
}

func (t *File) Sync() (e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	return t.file.Sync()
}

func (t *File) Create() {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}
}

func (t *File) Delete() error {
	if !t.l.TryLock() {
		return ErrFailToLock
	}
	defer t.l.Unlock()

	if t.IsDir() {
		return os.RemoveAll(t.Config.FilePath)
	}

	return os.Remove(t.Config.FilePath)
}

func (t *File) Close() error {
	if t.file != nil {
		if e := t.file.Close(); e != nil {
			return e
		} else {
			t.file = nil
		}
	}
	return nil
}

func (t *File) IsExist() bool {
	if len(t.Config.FilePath) > 4096 {
		panic(ErrFilePathTooLong)
	}

	_, err := os.Stat(t.Config.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		} else {
			if !strings.Contains(err.Error(), "file name too long") {
				panic(ErrFilePathTooLong)
			}
			return false
		}
	}
	return true
}

func (t *File) IsDir() bool {
	if len(t.Config.FilePath) > 4096 {
		panic(ErrFilePathTooLong)
	}

	info, err := os.Stat(t.Config.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		} else {
			if !strings.Contains(err.Error(), "file name too long") {
				panic(ErrFilePathTooLong)
			}
			return false
		}
	}
	return info.IsDir()
}

func (t *File) File() *os.File {
	t.getRWCloser()
	return t.file
}

func (t *File) getRWCloser() {
	if t.Config.AutoClose || t.file == nil {
		if !t.IsExist() {
			if e := t.newPath(); e != nil {
				panic(e)
			}
			if f, e := os.Create(t.Config.FilePath); e != nil {
				panic(e)
			} else {
				if t.Config.CurIndex > 0 {
					t.cu = t.Config.CurIndex
					t.cu, e = f.Seek(t.cu, 0)
					if e != nil {
						panic(e)
					}
				}
				t.file = f
			}
		} else {
			if f, e := os.OpenFile(t.Config.FilePath, os.O_RDWR|os.O_EXCL, 0644); e != nil {
				panic(e)
			} else {
				if t.Config.CurIndex != 0 {
					t.cu = t.Config.CurIndex
					whenc := 0
					if t.Config.CurIndex < 0 {
						t.cu = t.cu + 1
						whenc = 2
					}
					t.cu, e = f.Seek(t.cu, whenc)
					if e != nil {
						panic(e)
					}
				}
				t.file = f
			}
		}
	}
}

func (t *File) newPath() error {

	/*
		如果filename路径不存在，就新建它
	*/
	var exist func(string) bool = func(s string) bool {
		_, err := os.Stat(s)
		return err == nil || os.IsExist(err)
	}

	for i := 0; true; {
		a := strings.Index(t.Config.FilePath[i:], "/")
		if a == -1 {
			break
		}
		if a == 0 {
			a = 1
		} //bug fix 当绝对路径时开头的/导致问题
		i = i + a + 1
		if !exist(t.Config.FilePath[:i-1]) {
			err := os.Mkdir(t.Config.FilePath[:i-1], os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func transferIO(r io.Reader, w io.Writer, byteInSec int64) (e error) {
	if byteInSec > 0 {
		limit := l.New(1, "1s", "-1s")
		defer limit.Close()

		buf := make([]byte, byteInSec)
		for {
			n, err := r.Read(buf)
			if n != 0 {
				w.Write(buf[:n])
			} else if err != nil {
				e = err
				break
			}
			limit.TO()
		}
	} else {
		_, e = io.Copy(w, r)
	}

	return nil
}

func (t *File) write() io.Writer {
	if t.Config.AutoClose || t.wr == nil {
		t.wr = io.Writer(t.file)
		if t.Config.Coder != nil {
			t.wr = t.Config.Coder.NewEncoder().Writer(t.wr)
		}
	}
	return t.wr
}

func (t *File) read() io.Reader {
	if t.Config.AutoClose || t.rr == nil {
		t.rr = io.Reader(t.file)
		if t.Config.Coder != nil {
			t.rr = t.Config.Coder.NewDecoder().Reader(t.rr)
		}
	}
	return t.rr
}
