package part

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	pe "github.com/qydysky/part/errors"
	pio "github.com/qydysky/part/io"
	encoder "golang.org/x/text/encoding"
)

var (
	ErrPathEscapes      = errors.New("ErrPathEscapes")
	ErrCopy             = errors.New("ErrCopy")
	ErrGetRW            = errors.New("ErrGetRW")
	ErrFilePathTooLong  = errors.New("ErrFilePathTooLong")
	ErrNewFileCantSeed  = errors.New("ErrNewFileCantSeed")
	ErrFailToLock       = errors.New("ErrFailToLock")
	ErrMaxReadSizeReach = errors.New("ErrMaxReadSizeReach")
	ErrNoDir            = errors.New("ErrNoDir")
	ErrArg              = errors.New("ErrArg")
)

type fosi interface {
	// Close() error
	Create(name string) (*os.File, error)
	Lstat(name string) (fs.FileInfo, error)
	Mkdir(name string, perm fs.FileMode) error
	// Name() string
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	Remove(name string) error
	RemoveAll(path string) error
	Stat(name string) (fs.FileInfo, error)
}

type dos struct {
	// close    func() error
	create func(name string) (*os.File, error)
	lstat  func(name string) (fs.FileInfo, error)
	mkdir  func(name string, perm fs.FileMode) error
	// name     func() string
	open      func(name string) (*os.File, error)
	openFile  func(name string, flag int, perm fs.FileMode) (*os.File, error)
	remove    func(name string) error
	removeAll func(path string) error
	stat      func(name string) (fs.FileInfo, error)
}

func (t dos) Create(name string) (*os.File, error) {
	return t.create(name)
}

func (t dos) Lstat(name string) (fs.FileInfo, error) {
	return t.lstat(name)
}

func (t dos) Mkdir(name string, perm fs.FileMode) error {
	return t.mkdir(name, perm)
}

func (t dos) Open(name string) (*os.File, error) {
	return t.open(name)
}

func (t dos) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return t.openFile(name, flag, perm)
}

func (t dos) Remove(name string) error {
	return t.remove(name)
}

func (t dos) RemoveAll(path string) error {
	return t.removeAll(path)
}

func (t dos) Stat(name string) (fs.FileInfo, error) {
	return t.stat(name)
}

type File struct {
	Config Config
	file   *os.File
	wr     io.Writer
	rr     io.Reader
	sr     io.Seeker
	l      sync.RWMutex
}

type Config struct {
	root      string
	FilePath  string //文件路径
	CurIndex  int64  //初始化光标位置
	AutoClose bool   //自动关闭句柄

	// wrap with encoder
	//https://pkg.go.dev/golang.org/x/text/encoding#section-directories
	Coder encoder.Encoding
}

type FS interface {
	fs.FS
	OpenFile(name string) (*File, error)
}

type dirFS string

func (dir dirFS) Open(name string) (fs.File, error) {
	return dir.OpenFile(name)
}

func (dir dirFS) OpenFile(name string) (*File, error) {
	fullname, err := dir.join(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return Open(fullname).CheckRoot(string(dir)), nil
}

// join returns the path for name in dir.
func (dir dirFS) join(name string) (string, error) {
	if dir == "" {
		return "", errors.New("os: DirFS with empty root")
	}
	name, err := filepath.Localize(name)
	if err != nil {
		return "", fs.ErrInvalid
	}
	if os.IsPathSeparator(dir[len(dir)-1]) {
		return string(dir) + name, nil
	}
	return string(dir) + string(os.PathSeparator) + name, nil
}

func DirFS(dir string) FS {
	return dirFS(dir)
}

func Open(filePath string) *File {
	return NewNoClose(filePath)
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

func NewNoClose(filePath string) *File {
	return New(filePath, 0, false)
}

func (t *File) Open(childRefPath string) *File {
	return t.NewNoClose(childRefPath)
}

func (t *File) New(childRefPath string, curIndex int64, autoClose bool) *File {
	if filepath.IsAbs(childRefPath) {
		return New(childRefPath, curIndex, autoClose)
	} else {
		return New(filepath.Clean(t.File().Name()+string(os.PathSeparator)+childRefPath), curIndex, autoClose)
	}
}

func (t *File) NewNoClose(childRefPath string) *File {
	if filepath.IsAbs(childRefPath) {
		return New(childRefPath, 0, false)
	} else {
		return New(filepath.Clean(t.File().Name()+string(os.PathSeparator)+childRefPath), 0, false)
	}
}

func (t *File) CheckRoot(root string) *File {
	t.Config.root = root
	if rel, e := filepath.Rel(root, t.Config.FilePath); e != nil {
		panic(e)
	} else {
		t.Config.FilePath = rel
	}
	newPath(dos{
		create:    os.Create,
		lstat:     os.Lstat,
		mkdir:     os.Mkdir,
		open:      os.Open,
		openFile:  os.OpenFile,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
		stat:      os.Stat,
	}, t.Config.root+"/.t", fs.ModePerm|fs.ModeDir)
	return t
}

func (t *File) CopyTo(to *File, copyIOConfig pio.CopyConfig, tryLock bool) error {
	if e := t.getRWCloser(); e != nil {
		return pe.Join(ErrGetRW, e)
	}
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	if e := to.getRWCloser(); e != nil {
		return pe.Join(ErrGetRW, e)
	}
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

	if e := pio.Copy(t.read(), to.write(), copyIOConfig); e != nil {
		return pe.Join(ErrCopy, e)
	}
	return nil
}

func (t *File) CopyToIoWriter(to io.Writer, copyIOConfig pio.CopyConfig) error {
	if e := t.getRWCloser(); e != nil {
		return e
	}
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
	if e := t.getRWCloser(); e != nil {
		return e
	}
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
	if e := t.getRWCloser(); e != nil {
		return e
	}
	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	if e := to.getRWCloser(); e != nil {
		return e
	}
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

func (t *File) Write(data []byte) (int, error) {
	return t.WriteRaw(data, true)
}

func (t *File) WriteRaw(data []byte, tryLock bool) (int, error) {
	if e := t.getRWCloser(); e != nil {
		return 0, e
	}
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
	if e := t.getRWCloser(); e != nil {
		return 0, e
	}
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
	if e := t.getRWCloser(); e != nil {
		return 0, e
	}
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
	if e := t.getRWCloser(); e != nil {
		return nil, e
	}
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

// EOF will return
func (t *File) ReadAll(perReadSize int, maxReadSize int) (data []byte, e error) {
	if e := t.getRWCloser(); e != nil {
		return nil, e
	}
	if info, e := t.Stat(); e == nil && info.Size() > int64(maxReadSize) {
		return nil, ErrMaxReadSizeReach
	} else {
		data = make([]byte, info.Size())
	}
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
		total   int
		reader  = t.read()
	)

	for maxReadSize > total {
		n, e = reader.Read(tmpArea)

		if n == 0 && e != nil {
			break
		}

		copy(data[total:], tmpArea[:n])
		total += n
	}

	if maxReadSize <= total {
		return nil, ErrMaxReadSizeReach
	}

	data = data[:total]

	return
}

// EOF will return
func (t *File) ReadToBuf(buf *[]byte, perReadSize int, maxReadSize int) (e error) {
	if e := t.getRWCloser(); e != nil {
		return e
	}
	if info, e := t.Stat(); e == nil && info.Size() > int64(maxReadSize) {
		return ErrMaxReadSizeReach
	} else if info.Size() > int64(cap(*buf)) {
		*buf = append(*buf, make([]byte, info.Size()-int64(cap(*buf)))...)
	}
	clear(*buf)
	*buf = (*buf)[:cap(*buf)]

	if t.Config.AutoClose {
		defer t.Close()
	}

	if !t.l.TryRLock() {
		return ErrFailToLock
	}
	defer t.l.RUnlock()

	var (
		tmpArea = make([]byte, perReadSize)
		n       int
		total   int
		reader  = t.read()
	)

	for maxReadSize > total {
		n, e = reader.Read(tmpArea)

		if n == 0 && e != nil {
			break
		}

		copy((*buf)[total:], tmpArea[:n])
		total += n
	}

	if maxReadSize <= total {
		return ErrMaxReadSizeReach
	}

	*buf = (*buf)[:total]

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
	if e := t.getRWCloser(); e != nil {
		return e
	}
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
	if e := t.getRWCloser(); e != nil {
		return e
	}
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
	if e := t.getRWCloser(); e != nil {
		return e
	}
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
	if e := t.getRWCloser(); e != nil {
		return 0, e
	}
	if t.Config.AutoClose {
		defer t.Close()
	}

	return t.file.Seek(0, int(AtCurrent))
}

func (t *File) Create(mode ...fs.FileMode) {
	if e := t.getRWCloser(mode...); e != nil {
		panic(e)
	}
	if t.Config.AutoClose {
		defer t.Close()
	}
}

func (t *File) Delete() error {
	if !t.l.TryLock() {
		return ErrFailToLock
	}
	defer t.l.Unlock()

	fos, e := t.getOs()
	if e != nil {
		return e
	}

	e = t.Close()
	if e != nil {
		return e
	}

	if t.IsDir() {
		return fos.RemoveAll(t.Config.FilePath)
	}

	return fos.Remove(t.Config.FilePath)
}

func (t *File) CloseErr(err ...*error) {
	if t.file != nil {
		if e := t.file.Close(); e != nil && len(err) > 0 {
			*err[0] = e
		} else {
			t.file = nil
		}
	}
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
	if err != nil {
		if errors.Is(err, ErrFilePathTooLong) {
			panic(err)
		} else if strings.HasSuffix(err.Error(), "path escapes from parent") {
			panic(ErrPathEscapes)
		}
	}
	return err == nil
}

func IsExist(path string) bool {
	return Open(path).IsExist()
}

func (t *File) IsDir() bool {
	info, err := t.Stat()
	if err != nil {
		return false
	}
	return info.IsDir()
}

// filiter return true will not append to dirFiles
func (t *File) DirFilesRange(dropFiliter ...func(os.FileInfo) bool) iter.Seq[*File] {
	return func(yield func(*File) bool) {
		if fis, e := t.File().Readdir(0); e == nil {
			for i := 0; i < len(fis); i++ {
				if len(dropFiliter) == 0 || !dropFiliter[0](fis[i]) {
					if !yield(t.Open(fis[i].Name())) {
						break
					}
				}
			}
		}
	}
}

// filiter return true will not append to dirFiles
func (t *File) DirFiles(dropFiliter ...func(os.FileInfo) bool) (dirFiles []string, err error) {
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
			if len(dropFiliter) == 0 || !dropFiliter[0](fis[i]) {
				dirFiles = append(dirFiles, path.Clean(f.Name())+string(os.PathSeparator)+fis[i].Name())
			}
		}
	}
	return
}

func (t *File) SelfName() string {
	if e := t.getRWCloser(); e != nil {
		panic(e)
	}
	ls := strings.Split(t.file.Name(), string(os.PathSeparator))
	return ls[len(ls)-1]
}

func (t *File) File() *os.File {
	if e := t.getRWCloser(); e != nil {
		panic(e)
	}
	return t.file
}

func (t *File) Stat() (fs.FileInfo, error) {
	if len(t.Config.FilePath) > 4096 {
		return nil, ErrFilePathTooLong
	}

	fos, e := t.getOs()
	if e != nil {
		return nil, e
	}

	info, err := fos.Stat(t.Config.FilePath)
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
	fos, e := t.getOs()
	if e != nil {
		panic(e)
	}
	fi, e := fos.OpenFile(t.Config.FilePath, syscall.O_RDONLY|syscall.O_EXCL, 0)
	return e != nil && fi.Close() == nil
}

func (t *File) getOs() (fosi, error) {
	if t.Config.root != "" {
		if root, e := os.OpenRoot(t.Config.root); e != nil {
			return nil, e
		} else {
			return dos{
				create:    root.Create,
				lstat:     root.Lstat,
				mkdir:     root.Mkdir,
				open:      root.Open,
				openFile:  root.OpenFile,
				remove:    root.Remove,
				removeAll: os.RemoveAll,
				stat:      root.Stat,
			}, nil
		}
	}
	return dos{
		create:    os.Create,
		lstat:     os.Lstat,
		mkdir:     os.Mkdir,
		open:      os.Open,
		openFile:  os.OpenFile,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
		stat:      os.Stat,
	}, nil
}

func (t *File) getRWCloser(mode ...fs.FileMode) error {
	fmode := fs.ModePerm
	if len(mode) != 0 {
		fmode = mode[0]
	}
	if t.Config.AutoClose || t.file == nil {
		fos, e := t.getOs()
		if e != nil {
			return e
		}
		if !t.IsExist() {
			newPath(fos, t.Config.FilePath, fs.ModeDir|fmode)
			if t.IsDir() {
				if f, e := fos.OpenFile(t.Config.FilePath, os.O_RDONLY|os.O_EXCL, fmode); e != nil {
					return e
				} else {
					t.file = f
				}
			} else {
				if f, e := fos.OpenFile(t.Config.FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fmode); e != nil {
					return e
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
				if f, e := fos.OpenFile(t.Config.FilePath, os.O_RDONLY|os.O_EXCL, fmode); e != nil {
					return e
				} else {
					t.file = f
				}
			} else {
				if f, e := fos.OpenFile(t.Config.FilePath, os.O_RDWR|os.O_EXCL, fmode); e != nil {
					return e
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
							return e
						}
					}
					t.file = f
				}
			}
		}
	}
	return nil
}

func newPath(fos fosi, path string, mode fs.FileMode) {
	isDir := path[len(path)-1] == '/' || path[len(path)-1] == '\\'
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	_ = fos.Mkdir(filepath.Dir(filepath.Clean(path)), mode)
	if isDir {
		_ = fos.Mkdir(filepath.Clean(path)+string(os.PathSeparator), mode)
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
