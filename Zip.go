package part

import (
    "sync"
	"os"
    "github.com/klauspost/compress/zip"
	"io"
    "path/filepath"
    "strings"
    "bytes"
    "time"
)

type lzip struct {sync.Mutex}

func Zip() *lzip{
    return &lzip{}
}

func (this *lzip) InZip(srcFile string, destZip string) error {
    this.Lock()
	defer this.Unlock()

    zipfile, err := os.Create(destZip)
    if err != nil {
        return err
    }
    defer zipfile.Close()

    archive := zip.NewWriter(zipfile)
    defer archive.Close()

    if Sys().GetSys("windows") {srcFile=strings.Replace(srcFile,"/","\\",-1)}

    filepath.Walk(srcFile, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
		}
		
		var path_cut string = "/"
		if Sys().GetSys("windows") {path_cut="\\"}
		
		if path[len(path)-1:] == path_cut {return nil}

        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }
        
        header.Name = strings.TrimPrefix(path, filepath.Dir(srcFile) + path_cut)
        // header.Name = path
        if info.IsDir() {
            header.Name += path_cut
        } else {
            header.Method = zip.Deflate
        }

        writer, err := archive.CreateHeader(header)
        if err != nil {
            return err
        }

        if ! info.IsDir() {
            file, err := os.Open(path)
            if err != nil {
                return err
            }
            defer file.Close()
            _, err = io.Copy(writer, file)
        }
        return err
    })

    return err
}

func (this *lzip) UnZip(zipFile string, destDir string) error {
    this.Lock()
	defer this.Unlock()

    r, err := zip.OpenReader(zipFile)
    if err != nil {
        return err
    }
    defer func() {
        if err := r.Close(); err != nil {
            panic(err)
        }
    }()

    os.MkdirAll(destDir, 0755)

    // Closure to address file descriptors issue with all the deferred .Close() methods
    extractAndWriteFile := func(f *zip.File) error {
        rc, err := f.Open()
        if err != nil {
            return err
        }
        defer func() {
            if err := rc.Close(); err != nil {
                panic(err)
            }
        }()

        path := filepath.Join(destDir, f.Name)

        if f.FileInfo().IsDir() {
            os.MkdirAll(path, f.Mode())
        } else {
            os.MkdirAll(filepath.Dir(path), f.Mode())
            f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
            if err != nil {
                return err
            }
            defer func() {
                if err := f.Close(); err != nil {
                    panic(err)
                }
            }()

            _, err = io.Copy(f, rc)
            if err != nil {
                return err
            }
        }
        return nil
    }

    for _, f := range r.File {
        err := extractAndWriteFile(f)
        if err != nil {
            return err
        }
    }

    return nil
}

type rZip struct {
    poit *zip.ReadCloser
    buf map[string]*zip.File
}

func RZip() *rZip {return &rZip{}}

func (t *rZip) New(zipFile string) (error) {
    t.buf  = make(map[string]*zip.File)

    var err error
    t.poit, err = zip.OpenReader(zipFile)
    if err != nil {return err}

    for _, _f := range t.poit.File {
        if _f.FileInfo().IsDir() {continue}
        t.buf[_f.Name] = _f
    }
    
    return nil
}

func (t *rZip) Read(path string) (*bytes.Buffer,string,error) {
    var timeLayoutStr = "2006-01-02 15:04:05"
    var err error

    if f,ok := t.buf[path];ok {
        if rc, err := f.Open();err == nil {
            defer rc.Close();

            buf := new(bytes.Buffer)
            buf.ReadFrom(rc)
            return buf,f.FileHeader.Modified.Format(timeLayoutStr),nil
        }
        return &bytes.Buffer{},time.Now().UTC().Format(timeLayoutStr),err
    }
    return &bytes.Buffer{},time.Now().UTC().Format(timeLayoutStr),nil
}

func (t *rZip) Close() {
    t.poit.Close()
    for k := range t.buf {
        delete(t.buf, k)
    }
}