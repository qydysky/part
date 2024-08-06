package part

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func Md5String(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func Md5File(path string) (string, error) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return "", err
	}

	h := md5.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// func GetFileModTime(path string) (error, int64) {
// 	f, err := os.Open(path)
// 	if err != nil {
// 		fmt.Println("open file error")
// 		return err, time.Now().Unix()
// 	}
// 	defer f.Close()

// 	fi, err := f.Stat()
// 	if err != nil {
// 		fmt.Println("stat fileinfo error")
// 		return err, time.Now().Unix()
// 	}

// 	return nil, fi.ModTime().Unix()
// }
