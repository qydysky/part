package part 

import (
    "fmt"
	"time"
	"runtime"
	"strings"
	"syscall"
    "errors"
    "os"
	"io/ioutil"
)

type checkfile struct {}

func Checkfile() *checkfile{
    return &checkfile{}
}

func (this *checkfile) Build(checkFile,checkDir,SplitString string) {
	
	v,_,_:=this.GetAllFile(checkDir)
	_checkFile := Filel {
		File:checkFile,
		Write:true,
		Loc:0,
	}

	Logf().I("checkFile Build:begin")

	for _,value := range v {
		_checkFile.Context += value + SplitString
	}
	
	File().FileWR(_checkFile)
	Logf().I("checkFile Build:ok")

}

func (this *checkfile) IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func (this *checkfile) IsOpen(f string) bool {
	fi,e:=os.OpenFile(f, syscall.O_RDONLY|syscall.O_EXCL, 0)
	if e!=nil {return true}
	fi.Close()
	return false
}

func (this *checkfile) Checkfile(src string)(string,error){

    _,str,err:=this.GetAllFile(src)

    if err !=nil {return "",errors.New("获取文件列表错误！")}

    return Md5().Md5String(str),nil

}

func (this *checkfile) GetAllFile(pathname string) ([]string,string,error) {

    var (
		returnVal string = ""
		list []string
	)

    rd, err := ioutil.ReadDir(pathname)

    for _, fi := range rd {
        if fi.IsDir() {
            _list, _returnVal, _:=this.GetAllFile(pathname + fi.Name() + "/")
			returnVal+=_returnVal
			list = append(list, _list...)
        } else {
			returnVal+=pathname + "/" + fi.Name() + "\n"
			list = append(list, pathname + fi.Name())
        }
    }
    return list,returnVal,err
}

func (this *checkfile) GetFileSize(path string) int64 {

    if !this.IsExist(path) {
        return 0
    }
    fileInfo, err := os.Stat(path)
    if err != nil {
        return 0
    }
    return fileInfo.Size()
}

func (this *checkfile) CheckList(checkFile,SplitString string)bool{
	
	if checkFile == "" || SplitString == "" {
		Logf().E("[err]checkFile or SplitString has null.")
		return false
	}
	Logf().I("===checkFile Check===")

	var checkFileString string
	var checkFileList []string 
	if strings.Contains(checkFile,"https://") {
		Logf().I("[wait]checkFile: Getting checkfile...")

		var r = ReqfVal {
			Url:checkFile,
			Timeout:6,
            Retry:2,
		}
		
		b,_,e:=Reqf(r)
		if e != nil {
			Logf().E("[err]checkFile:",checkFile,e.Error())
			return false
		}else{
			Logf().I("[ok]checkFile: Get checkfile.")
			checkFileString=string(b)
		}
	}else{
		var _checkFile = Filel {
			File:checkFile,
			Write:false,
			Loc:0,
		}
		
		checkFileString=File().FileWR(_checkFile)
	}

	checkFileList=strings.Split(checkFileString,SplitString);

	var returnVal bool = true
	for _,value := range checkFileList {
		if value!=""&&!this.IsExist(value) {
			Logf().E("[err]checkFile:",value,"not exist!")
			returnVal=false
		}else{
			if runtime.GOOS!="windows" && strings.Contains(value,".run") {
				var want os.FileMode = 0700
				if !this.CheckFilePerm(value,want) {
					Logf().E("[err]checkFile:",value,"no permission!")
					returnVal=false
				}
			}
			// fmt.Println("[ok]checkFile:",checkDir+value)
		}
	}
	if returnVal {Logf().I("[ok]checkFile: all file pass!")}
	Logf().I("===checkFile Check===")

	return returnVal
}

func (this *checkfile) GetFileModTime(path string) (error,int64) {

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

func (this *checkfile) CheckFilePerm(f string,want os.FileMode)bool{
	fi, err := os.Lstat(f)
	if err != nil {
		Logf().E("[err]cant get permission : ",f);
		return false
	}
	return fi.Mode().Perm()>=want
}
