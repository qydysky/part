package part

// import (
// 	"errors"
// 	"fmt"
// 	"os"
// 	"runtime"
// 	"strings"
// 	"syscall"
// 	"time"

// 	reqf "github.com/qydysky/part/reqf"
// )

// type checkfile struct {
// 	RV []interface{}
// }

// func Checkfile() *checkfile {
// 	return &checkfile{}
// }

// func (t *checkfile) Build(checkFile, root, checkDir, SplitString string, usemd5 bool) {

// 	v, _, _ := t.GetAllFile(checkDir)
// 	_checkFile := Filel{
// 		File: checkFile,
// 		Loc:  0,
// 	}

// 	Logf().I("checkFile Build:begin")

// 	if usemd5 {
// 		_checkFile.Context = append(_checkFile.Context, "UseMd5")
// 	}

// 	_checkFile.Context = append(_checkFile.Context, SplitString)

// 	for _, value := range v {
// 		if usemd5 {
// 			md5, e := Md5().Md5File(value)
// 			if e != nil {
// 				md5 = "00000000000000000000000000000000"
// 			}
// 			_checkFile.Context = append(_checkFile.Context, md5)
// 		}
// 		_checkFile.Context = append(_checkFile.Context, value[len(root):])
// 		_checkFile.Context = append(_checkFile.Context, SplitString)
// 	}

// 	File().FileWR(_checkFile)
// 	Logf().I("checkFile Build:ok")

// }

// func (t *checkfile) IsExist(f string) bool {
// 	if len(f) > 4096 {
// 		return false
// 	}

// 	_, err := os.Stat(f)
// 	if err != nil {
// 		if errors.Is(err, os.ErrNotExist) {
// 			t.RV = append(t.RV, false, nil)
// 			return false
// 		} else {
// 			if !strings.Contains(err.Error(), "file name too long") {
// 				Logf().E(err)
// 			}
// 			t.RV = append(t.RV, false, err)
// 			return false
// 		}
// 	}
// 	t.RV = append(t.RV, true, nil)
// 	return true
// }

// func (t *checkfile) IsOpen(f string) bool {
// 	if !t.IsExist(f) {
// 		return false
// 	}

// 	fi, e := os.OpenFile(f, syscall.O_RDONLY|syscall.O_EXCL, 0)
// 	if e != nil {
// 		return true
// 	}
// 	fi.Close()
// 	return false
// }

// func (t *checkfile) Checkfile(src string) (string, error) {

// 	_, str, err := t.GetAllFile(src)

// 	if err != nil {
// 		return "", errors.New("获取文件列表错误！")
// 	}

// 	return Md5().Md5String(str), nil

// }

// func (t *checkfile) GetAllFile(pathname string) ([]string, string, error) {

// 	var (
// 		returnVal string = ""
// 		list      []string
// 	)

// 	rd, err := os.ReadDir(pathname)

// 	if err != nil {
// 		return list, returnVal, err
// 	}

// 	for _, fi := range rd {
// 		if fi.IsDir() {
// 			_list, _returnVal, _ := t.GetAllFile(pathname + fi.Name() + "/")
// 			returnVal += _returnVal
// 			list = append(list, _list...)
// 		} else {
// 			returnVal += pathname + "/" + fi.Name() + "\n"
// 			list = append(list, pathname+fi.Name())
// 		}
// 	}
// 	return list, returnVal, err
// }

// func (t *checkfile) GetFileSize(path string) int64 {

// 	if !t.IsExist(path) {
// 		return 0
// 	}
// 	fileInfo, err := os.Stat(path)
// 	if err != nil {
// 		return 0
// 	}
// 	return fileInfo.Size()
// }

// func (t *checkfile) CheckList(checkFile, root, SplitString string) bool {

// 	if checkFile == "" || SplitString == "" {
// 		Logf().E("[err]checkFile or SplitString has null.")
// 		return false
// 	}
// 	Logf().I("===checkFile Check===")

// 	var checkFileString string
// 	var checkFileList []string
// 	if strings.Contains(checkFile, "https://") {
// 		Logf().I("[wait]checkFile: Getting checkfile...")

// 		var r = reqf.Rval{
// 			Url:     checkFile,
// 			Timeout: 6,
// 			Retry:   2,
// 		}
// 		req := reqf.New()
// 		if e := req.Reqf(r); e != nil {
// 			Logf().E("[err]checkFile:", checkFile, e.Error())
// 			return false
// 		} else {
// 			Logf().I("[ok]checkFile: Get checkfile.")
// 			checkFileString = string(req.Respon)
// 		}
// 	} else {
// 		var _checkFile = Filel{
// 			File: checkFile,
// 			Loc:  0,
// 		}

// 		checkFileString = File().FileWR(_checkFile)
// 	}

// 	checkFileList = strings.Split(checkFileString, SplitString)

// 	var (
// 		returnVal bool = true
// 		UseMd5    bool = strings.Contains(checkFileList[0], "UseMd5")
// 		_value    string
// 	)

// 	for _, value := range checkFileList[1:] {
// 		if value == "" {
// 			continue
// 		}

// 		if UseMd5 {
// 			_value = root + value[32:]
// 		} else {
// 			_value = root + value
// 		}

// 		if !t.IsExist(_value) {
// 			Logf().E("[err]checkFile:", _value, "not exist!")
// 			returnVal = false
// 		} else {
// 			if UseMd5 {
// 				if md5, _ := Md5().Md5File(_value); value[:32] != md5 {
// 					Logf().E("[err]checkFile:", _value, "no match!")
// 					returnVal = false
// 				}
// 			}

// 			if runtime.GOOS != "windows" && strings.Contains(_value, ".run") {
// 				var want os.FileMode = 0700
// 				if !t.CheckFilePerm(_value, want) {
// 					Logf().E("[err]checkFile:", _value, "no permission!")
// 					returnVal = false
// 				}
// 			}
// 			// fmt.Println("[ok]checkFile:",checkDir+value)
// 		}

// 	}
// 	if returnVal {
// 		Logf().I("[ok]checkFile: all file pass!")
// 	}
// 	Logf().I("===checkFile Check===")

// 	return returnVal
// }

// func (t *checkfile) GetFileModTime(path string) (error, int64) {

// 	if !t.IsExist(path) {
// 		return errors.New("not exist"), time.Now().Unix()
// 	}

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

// func (t *checkfile) CheckFilePerm(f string, want os.FileMode) bool {
// 	fi, err := os.Lstat(f)
// 	if err != nil {
// 		Logf().E("[err]cant get permission : ", f)
// 		return false
// 	}
// 	return fi.Mode().Perm() >= want
// }
