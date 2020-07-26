package part

import (
    "sync"
	"os"
    "github.com/klauspost/compress/zip"
	"io"
    "path/filepath"
	"strings"
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