package part

import (
    // "sync"
	// "os"
    "github.com/klauspost/compress/flate"
	// "io"
    // "path/filepath"
    // "strings"
    "bytes"
    // "time"
    // "errors"
)

type lflate struct {}

func Flate() *lflate{
    return &lflate{}
}

func (this *lflate) InFlate(byteS []byte, level int) ([]byte,error) {
    buf := bytes.NewBuffer(nil)

    // 创建一个flate.Write
    flateWrite, err := flate.NewWriter(buf, level)
    if err != nil {
        Logf().E(err.Error())
        return buf.Bytes(),err
    }
    defer flateWrite.Close()
    // 写入待压缩内容
    flateWrite.Write(byteS)
    flateWrite.Flush()
    return buf.Bytes(),nil
}

// func (this *lflate) UnFlate(zipFile string, destDir string) error {
//     this.Lock()
// 	defer this.Unlock()

//     r, err := zip.OpenReader(zipFile)
//     if err != nil {
//         return err
//     }
//     defer func() {
//         if err := r.Close(); err != nil {
//             panic(err)
//         }
//     }()

//     os.MkdirAll(destDir, 0755)

//     // Closure to address file descriptors issue with all the deferred .Close() methods
//     extractAndWriteFile := func(f *zip.File) error {
//         rc, err := f.Open()
//         if err != nil {
//             return err
//         }
//         defer func() {
//             if err := rc.Close(); err != nil {
//                 panic(err)
//             }
//         }()

//         path := filepath.Join(destDir, f.Name)

//         if f.FileInfo().IsDir() {
//             os.MkdirAll(path, f.Mode())
//         } else {
//             os.MkdirAll(filepath.Dir(path), f.Mode())
//             f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
//             if err != nil {
//                 return err
//             }
//             defer func() {
//                 if err := f.Close(); err != nil {
//                     panic(err)
//                 }
//             }()

//             _, err = io.Copy(f, rc)
//             if err != nil {
//                 return err
//             }
//         }
//         return nil
//     }

//     for _, f := range r.File {
//         err := extractAndWriteFile(f)
//         if err != nil {
//             return err
//         }
//     }

//     return nil
// }

// type rZip struct {
//     poit *zip.ReadCloser
//     buf map[string]*zip.File
//     sync.Mutex
// }

// func RZip() *rZip {return &rZip{}}

// func (t *rZip) New(zipFile string) (error) {
//     t.Lock()
// 	defer t.Unlock()

//     t.buf  = make(map[string]*zip.File)

//     var err error
//     t.poit, err = zip.OpenReader(zipFile)
//     if err != nil {return err}

//     for _, _f := range t.poit.File {
//         if _f.FileInfo().IsDir() {continue}
//         t.buf[_f.Name] = _f
//     }
    
//     return nil
// }

// func (t *rZip) Read(path string) (*bytes.Buffer,string,error) {
//     t.Lock()
//     defer t.Unlock()
    
//     var timeLayoutStr = "2006-01-02 15:04:05"
//     var err error

//     if f,ok := t.buf[path];ok {
//         if rc, err := f.Open();err == nil {
//             defer rc.Close();

//             buf := new(bytes.Buffer)
//             buf.ReadFrom(rc)
//             return buf,f.FileHeader.Modified.Format(timeLayoutStr),nil
//         }
//         return &bytes.Buffer{},time.Now().UTC().Format(timeLayoutStr),err
//     }
//     return &bytes.Buffer{},time.Now().UTC().Format(timeLayoutStr),errors.New("not found")
// }

// func (t *rZip) List() []string {
//     var list []string
//     for k := range t.buf {
//         list=append(list,k)
//     }
//     return list
// }

// func (t *rZip) Close() {
//     t.poit.Close()
//     for k := range t.buf {
//         delete(t.buf, k)
//     }
// }