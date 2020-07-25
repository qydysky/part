package part

import (
    "sync"
    "io"
	"crypto/md5"
	"fmt"
    "os"
    "time"
)

type md5l struct{sync.Mutex}

func Md5() *md5l{
    return &md5l{}
}

func (this *md5l) Md5String(str string) string {
    this.Lock()
	defer this.Unlock()

    w := md5.New()
    io.WriteString(w, str)
    md5str := fmt.Sprintf("%x", w.Sum(nil))
    return md5str
}

func (this *md5l) Md5File(path string) (string, error) {
    this.Lock()
	defer this.Unlock()

    file, err := os.Open(path)
    defer file.Close() 
    if err != nil {
        return "",err 
    }

    h := md5.New()
    _, err = io.Copy(h,file)
    if err != nil {
        return "",err 
    }

    return fmt.Sprintf("%x",h.Sum(nil)), nil 
}

func (this *md5l) GetFileModTime(path string) (error,int64) {
    this.Lock()
	defer this.Unlock()

    f, err := os.Open(path)
	if err != nil {
		fmt.Println("open file error")
		return err,time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fmt.Println("stat fileinfo error")
		return err,time.Now().Unix()
	}

	return nil,fi.ModTime().Unix()
}
