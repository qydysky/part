package part

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	l "github.com/qydysky/part/limit"
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
	sync.RWMutex
}

type Config struct {
	FilePath  string //文件路径
	CurIndex  int64  //初始化光标位置
	AutoClose bool   //自动关闭句柄
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

	if !t.TryRLock() {
		return ErrFailToLock
	}
	defer t.RUnlock()

	to.getRWCloser()
	if t.Config.AutoClose {
		defer to.Close()
	}

	if tryLock {
		if !to.TryLock() {
			return ErrFailToLock
		}
	} else {
		to.Lock()
	}
	defer to.Unlock()

	return transfer(t.file, to.file, byteInSec)
}

func (t *File) Write(data []byte, tryLock bool) (int, error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if tryLock {
		if !t.TryLock() {
			return 0, ErrFailToLock
		}
	} else {
		t.Lock()
	}
	defer t.Unlock()

	return t.file.Write(data)
}

func (t *File) Read(data []byte) (int, error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.TryRLock() {
		return 0, ErrFailToLock
	}
	defer t.RUnlock()

	return t.file.Read(data)
}

func (t *File) ReadUntil(separation byte, perReadSize int, maxReadSize int) (data []byte, e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.TryRLock() {
		return nil, ErrFailToLock
	}
	defer t.RUnlock()

	var (
		tmpArea = make([]byte, perReadSize)
		n       int
	)

	for maxReadSize > 0 {
		n, e = t.file.Read(tmpArea)

		if e != nil {
			if errors.Is(e, io.EOF) {
				e = nil
			}
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

func (t *File) Seed(index int64) (e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.TryLock() {
		return ErrFailToLock
	}
	defer t.Unlock()

	whenc := 0
	if index < 0 {
		whenc = 2
	}
	t.Config.CurIndex, e = t.file.Seek(index, whenc)

	return nil
}

func (t *File) Delete() error {
	if !t.TryLock() {
		return ErrFailToLock
	}
	defer t.Unlock()

	return os.Remove(t.Config.FilePath)
}

func (t *File) Close() error {
	if t.file != nil {
		return t.file.Close()
	}
	return nil
}

func (t *File) getRWCloser() {
	if t.Config.AutoClose || t.file == nil {
		if !t.isExist() {
			if f, e := os.Create(t.Config.FilePath); e != nil {
				panic(e)
			} else {
				if t.Config.CurIndex != 0 {
					whenc := 0
					if t.Config.CurIndex < 0 {
						whenc = 2
					}
					t.Config.CurIndex, e = f.Seek(t.Config.CurIndex, whenc)
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
					whenc := 0
					if t.Config.CurIndex < 0 {
						whenc = 2
					}
					t.Config.CurIndex, e = f.Seek(t.Config.CurIndex, whenc)
					if e != nil {
						panic(e)
					}
				}
				t.file = f
			}
		}
	}
}

func transfer(r io.ReadCloser, w io.WriteCloser, byteInSec int64) (e error) {
	if byteInSec > 0 {
		limit := l.New(1, 1000, -1)
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

func (t *File) isExist() bool {
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
