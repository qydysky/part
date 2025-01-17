package part

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	pio "github.com/qydysky/part/io"
	encoder "golang.org/x/text/encoding"
)

var (
	ErrFilePathTooLong  = errors.New("ErrFilePathTooLong")
	ErrNewFileCantSeed  = errors.New("ErrNewFileCantSeed")
	ErrFailToLock       = errors.New("ErrFailToLock")
	ErrMaxReadSizeReach = errors.New("ErrMaxReadSizeReach")
	ErrNoDir            = errors.New("ErrNoDir")
	ErrArg              = errors.New("ErrArg")
)

type File struct {
	Config Config
	file   *os.File
	wr     io.Writer
	rr     io.Reader
	sr     io.Seeker
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

func (t *File) CopyTo(to *File, copyIOConfig pio.CopyConfig, tryLock bool) error {
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

	return pio.Copy(t.read(), to.write(), copyIOConfig)
}

func (t *File) CopyToIoWriter(to io.Writer, copyIOConfig pio.CopyConfig) error {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	return pio.Copy(t.read(), to, copyIOConfig)
}

func (t *File) CopyFromIoReader(from io.Reader, copyIOConfig pio.CopyConfig) error {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	return pio.Copy(from, t.write(), copyIOConfig)
}

// stop after untilBytes
//
// data not include untilBytes
func (t *File) CopyToUntil(to *File, untilBytes []byte, perReadSize int, maxReadSize int, tryLock bool) (e error) {
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

	var (
		reserve = len(untilBytes) - 1
		tmpArea = make([]byte, reserve+perReadSize)
		n       int
		reader  = t.read()
	)

	{
		var seekN int
		if reserve != 0 {
			//avoid spik
			if _, e := t.file.Seek(-int64(reserve), int(AtCurrent)); e == nil {
				seekN = reserve
			}
		}
		n, e = reader.Read(tmpArea)
		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:n], untilBytes); i != -1 {
			if n-i-len(untilBytes) != 0 {
				_, _ = t.file.Seek(-int64(n-i-len(untilBytes)), int(AtCurrent))
			}
			if i != 0 {
				if _, e := to.file.Write(tmpArea[seekN:i]); e != nil {
					return e
				}
			}
			return
		} else {
			if _, e := to.file.Write(tmpArea[seekN:n]); e != nil {
				return e
			}
		}
	}

	for maxReadSize > 0 {
		if reserve != 0 {
			copy(tmpArea, tmpArea[reserve:])
		}
		n, e = reader.Read(tmpArea[reserve:])

		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:reserve+n], untilBytes); i != -1 {
			if reserve+n-i-len(untilBytes) != 0 {
				_, _ = t.file.Seek(-int64(reserve+n-i-len(untilBytes)), int(AtCurrent))
			}
			if i != 0 {
				if _, e := to.file.Write(tmpArea[reserve:i]); e != nil {
					return e
				}
			}
			break
		} else {
			if _, e := to.file.Write(tmpArea[reserve:n]); e != nil {
				return e
			}
		}
	}

	if maxReadSize <= 0 {
		e = ErrMaxReadSizeReach
	}

	return
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

func (t *File) Seek(offset int64, whence int) (int64, error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return 0, ErrFailToLock
	}
	defer t.l.RUnlock()

	return t.seek().Seek(offset, whence)
}

// stop after untilBytes
//
// data not include untilBytes
func (t *File) ReadUntil(untilBytes []byte, perReadSize int, maxReadSize int) (data []byte, e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return nil, ErrFailToLock
	}
	defer t.l.RUnlock()

	var (
		reserve = len(untilBytes) - 1
		tmpArea = make([]byte, reserve+perReadSize)
		n       int
		reader  = t.read()
	)

	{
		var seekN int
		if reserve != 0 {
			//avoid spik
			if _, e := t.file.Seek(-int64(reserve), int(AtCurrent)); e == nil {
				seekN = reserve
			}
		}
		n, e = reader.Read(tmpArea)
		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:n], untilBytes); i != -1 {
			if n-i-len(untilBytes) != 0 {
				_, _ = t.file.Seek(-int64(n-i-len(untilBytes)), int(AtCurrent))
			}
			if i != 0 {
				data = append(data, tmpArea[seekN:i]...)
			}
			return
		} else {
			data = append(data, tmpArea[seekN:n]...)
		}
	}

	for maxReadSize > 0 {
		if reserve != 0 {
			copy(tmpArea, tmpArea[reserve:])
		}
		n, e = reader.Read(tmpArea[reserve:])

		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:reserve+n], untilBytes); i != -1 {
			if reserve+n-i-len(untilBytes) != 0 {
				_, _ = t.file.Seek(-int64(reserve+n-i-len(untilBytes)), int(AtCurrent))
			}
			if i != 0 {
				data = append(data, tmpArea[reserve:i]...)
			}
			break
		} else {
			data = append(data, tmpArea[reserve:n]...)
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
		n       int
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

type FileWhence int

const (
	AtOrigin FileWhence = iota
	AtCurrent
	AtEnd
)

// Seek sets the offset for the next Read or Write on file to offset
func (t *File) SeekIndex(index int64, whence FileWhence) (e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryLock() {
		return ErrFailToLock
	}
	defer t.l.Unlock()

	_, e = t.file.Seek(index, int(whence))

	return
}

// stop before untilBytes
func (t *File) SeekUntil(untilBytes []byte, whence FileWhence, perReadSize int, maxReadSize int) (e error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	if whence == AtOrigin {
		_, _ = t.file.Seek(0, int(whence))
	}

	var (
		reserve = len(untilBytes) - 1
		tmpArea = make([]byte, reserve+perReadSize)
		n       int
		reader  = t.read()
	)

	if reserve != 0 {
		//avoid spik
		_, _ = t.file.Seek(-int64(reserve), int(AtCurrent))
	}

	{
		n, e = reader.Read(tmpArea)
		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:n], untilBytes); i != -1 {
			if n-i != 0 {
				_, _ = t.file.Seek(-int64(n-i), int(AtCurrent))
			}
			return
		}
	}

	for maxReadSize > 0 {
		if reserve != 0 {
			copy(tmpArea, tmpArea[reserve:])
		}
		n, e = reader.Read(tmpArea[reserve:])
		if n == 0 && e != nil {
			return
		}

		maxReadSize = maxReadSize - n

		if i := bytes.Index(tmpArea[:reserve+n], untilBytes); i != -1 {
			if reserve+n-i != 0 {
				_, _ = t.file.Seek(-int64(reserve+n-i), int(AtCurrent))
			}
			break
		}
	}

	if maxReadSize <= 0 {
		e = ErrMaxReadSizeReach
	}

	return
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

func (t *File) CurIndex() (ret int64, err error) {
	t.getRWCloser()
	if t.Config.AutoClose {
		defer t.Close()
	}

	return t.file.Seek(0, int(AtCurrent))
}

func (t *File) Create(mode ...fs.FileMode) {
	t.getRWCloser(mode...)
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
	_, err := t.Stat()
	if errors.Is(err, ErrFilePathTooLong) {
		panic(err)
	}
	return err == nil
}

func (t *File) IsDir() bool {
	info, err := t.Stat()
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (t *File) DirFiles() (files []string, err error) {
	if !t.IsDir() {
		err = ErrNoDir
		return
	}

	f := t.File()
	if fis, e := f.Readdir(0); e != nil {
		err = e
		return
	} else {
		for i := 0; i < len(fis); i++ {
			files = append(files, path.Clean(f.Name())+string(os.PathSeparator)+fis[i].Name())
		}
	}
	return
}

func (t *File) File() *os.File {
	t.getRWCloser()
	return t.file
}

func (t *File) Stat() (fs.FileInfo, error) {
	if len(t.Config.FilePath) > 4096 {
		return nil, ErrFilePathTooLong
	}

	info, err := os.Stat(t.Config.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		} else {
			if !strings.Contains(err.Error(), "file name too long") {
				return nil, ErrFilePathTooLong
			}
			return nil, err
		}
	}
	return info, nil
}

func (t *File) GetFileModTimeT() (mod time.Time, err error) {
	fi, err := t.Stat()
	if err != nil {
		return time.Now(), err
	}
	return fi.ModTime(), nil
}

func (t *File) GetFileModTime() (mod int64, err error) {
	fi, err := t.Stat()
	if err != nil {
		return -1, err
	}
	return fi.ModTime().Unix(), nil
}

func (t *File) GetFileSize() (int64, error) {
	fi, err := t.Stat()
	if err != nil {
		return -1, err
	}
	return fi.Size(), nil
}

func (t *File) HavePerm(want os.FileMode) (bool, error) {
	fi, err := t.Stat()
	if err != nil {
		return false, err
	}
	return fi.Mode().Perm() >= want, nil
}

func (t *File) IsOpen() bool {
	if !t.IsExist() {
		return false
	}
	fi, e := os.OpenFile(t.Config.FilePath, syscall.O_RDONLY|syscall.O_EXCL, 0)
	defer fi.Close()
	return e != nil
}

func (t *File) getRWCloser(mode ...fs.FileMode) {
	fmode := fs.ModePerm
	if len(mode) != 0 {
		fmode = mode[0]
	}
	if t.Config.AutoClose || t.file == nil {
		if !t.IsExist() {
			newPath(t.Config.FilePath, fs.ModeDir|fmode)
			if t.IsDir() {
				if f, e := os.OpenFile(t.Config.FilePath, os.O_RDONLY|os.O_EXCL, fmode); e != nil {
					panic(e)
				} else {
					t.file = f
				}
			} else {
				if f, e := os.OpenFile(t.Config.FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fmode); e != nil {
					panic(e)
				} else {
					// if t.Config.CurIndex > 0 {
					// 	_, e = f.Seek(t.Config.CurIndex, int(AtOrigin))
					// 	if e != nil {
					// 		panic(e)
					// 	}
					// }
					t.file = f
				}
			}
		} else {
			if t.IsDir() {
				if f, e := os.OpenFile(t.Config.FilePath, os.O_RDONLY|os.O_EXCL, fmode); e != nil {
					panic(e)
				} else {
					t.file = f
				}
			} else {
				if f, e := os.OpenFile(t.Config.FilePath, os.O_RDWR|os.O_EXCL, fmode); e != nil {
					panic(e)
				} else {
					if t.Config.CurIndex != 0 {
						cu := t.Config.CurIndex
						whenc := AtOrigin
						if t.Config.CurIndex < 0 {
							cu += 1
							whenc = AtEnd
						}
						_, e = f.Seek(cu, int(whenc))
						if e != nil {
							panic(e)
						}
					}
					t.file = f
				}
			}
		}
	}
}

func newPath(path string, mode fs.FileMode) {
	rawPath := ""
	if !filepath.IsAbs(path) {
		rawPath, _ = os.Getwd()
	}
	rawPs := strings.Split(path, string(os.PathSeparator))
	for n, p := range rawPs {
		if p == "" || p == "." {
			continue
		}
		if n == len(rawPs)-1 {
			break
		}
		rawPath += string(os.PathSeparator) + p
		if _, err := os.Stat(rawPath); os.IsNotExist(err) {
			_ = os.Mkdir(rawPath, mode)
		}
	}
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

func (t *File) seek() io.Seeker {
	if t.Config.AutoClose || t.sr == nil {
		t.sr = io.Seeker(t.file)
		if t.Config.Coder != nil {
			panic("no support")
		}
	}
	return t.sr
}
