package part

import (
	"sync"
	"strings"
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"syscall"

	Ppart "github.com/qydysky/part/linuxwin"
)

type file struct {
	sync.Mutex
	F Filel
}

const (
	o_RDONLY int = syscall.O_RDONLY // 只读打开文件和os.Open()同义
	o_WRONLY int = syscall.O_WRONLY // 只写打开文件
	o_RDWR   int = syscall.O_RDWR   // 读写方式打开文件
	o_APPEND int = syscall.O_APPEND // 当写的时候使用追加模式到文件末尾
	o_CREATE int = syscall.O_CREAT  // 如果文件不存在，此案创建
	o_EXCL   int = syscall.O_EXCL   // 和o_CREATE一起使用, 只有当文件不存在时才创建
	o_SYNC   int = syscall.O_SYNC   // 以同步I/O方式打开文件，直接写入硬盘.
	o_TRUNC  int = syscall.O_TRUNC  // 如果可以的话，当打开文件时先清空文件
)

type Filel struct {
	File string //src
	Write bool //false:read
	Loc int64 //WriteOrRead loc ;0:rewrite Or read all;-1 write on end
	ReadNum int64
	Context []interface{} //Write string
}

func File() *file{
	return &file{}
}

func (this *file) FileWR(C Filel) string {
    this.Lock()
	defer this.Unlock()

	var returnVal string

	if C.File == "" {returnVal="";return returnVal}
	
	if C.Write {
		if len(C.Context) == 0 {return ""}

		switch C.Context[0].(type) {
		case io.Reader:
			if len(C.Context) != 1 {
				fmt.Println("Err:copy only allow one context")
				return ""
			}
			returnVal=this.copy(C)
		default:
			returnVal=this.write(C)
		}
	}else{
		returnVal=this.read(C)
	}

	return returnVal
}

func (this *file) copy(C Filel) string {
	var (
		File string=C.File
	)

	this.NewPath(File)

	fileObj,err := os.OpenFile(File,os.O_RDWR|os.O_EXCL|os.O_TRUNC,0644)
	if err != nil {
		fmt.Println("Err:cant open file:",File,err);
		return ""
	}
	defer fileObj.Close()

	if _, err := io.Copy(fileObj, C.Context[0].(io.Reader)); err != nil {
		fmt.Println("Err:cant copy file:",File,err);
		return ""
	}
	return "ok"
}

func (this *file) write(C Filel) string {

	var (
		File string=C.File
		Loc int64=C.Loc
	)

	this.NewPath(File)

	var Kind int 
	switch Loc {
		case 0:Kind=os.O_RDWR|os.O_EXCL|os.O_TRUNC
		default:Kind=os.O_RDWR|os.O_EXCL
	}

	fileObj,err := os.OpenFile(File,Kind,0644)
	if err != nil {
		fmt.Println("Err:cant open file:",File,err);
		return ""
	}
	defer fileObj.Close()

	Loc=this.locfix(Loc,File,fileObj)

	for _,v := range C.Context{
		switch v.(type) {
		case []uint8:
			tmp := v.([]byte)
			_, err = fileObj.WriteAt(tmp, Loc)
			if err != nil {
				fmt.Println("Err:cant write file:",File,err);
				return ""
			}
			Loc += int64(len(tmp))
		case string:
			tmp := []byte(v.(string))
			_, err = fileObj.WriteAt(tmp, Loc)
			if err != nil {
				fmt.Println("Err:cant write file:",File,err);
				return ""
			}
			Loc += int64(len(tmp))
		default:
			fmt.Println("Err:need context type string or []byte");
			return ""
		}
	}

	return "ok"
}

func (this *file) read(C Filel) string {

	var (
		File string=C.File
		Loc int64=C.Loc
		ReadNum int64=C.ReadNum
	)

	fileObj,err := os.OpenFile(File,os.O_RDONLY,0644)
	if err != nil {
		fmt.Println("Err:cant open file:",File,err);
		return ""
	}
	defer fileObj.Close()

	Loc=this.locfix(Loc,File,fileObj)

	if ReadNum == 0 {
		returnVal,err := ioutil.ReadAll(fileObj)

		if err != nil {
			fmt.Println("Err:cant read file:",File,err);
			return ""
		}

		return string(returnVal[Loc:])

	}
	
	buf := make([]byte, ReadNum)

	_, err=fileObj.ReadAt(buf,Loc)
	if err != nil {
		fmt.Println("Err:cant read file:",File,err);
		return ""
	}

	return string(buf)
}

func (this *file) locfix(Loc int64,File string,fileObj *os.File)int64{

	var returnVal int64

	FileInfo,err:=fileObj.Stat()
	if err != nil {
		fmt.Println("Err:cant read file lenght",File,err)
		return 0
	}

	if Loc<0 {
		returnVal=FileInfo.Size()+1+Loc
	}

	if returnVal<0 || returnVal>FileInfo.Size() {
		fmt.Println("Err:outrage of file lenght",File,Loc,"out of 0 ~",FileInfo.Size())
		return 0
	}

	return returnVal
}

func (this *file) NewPath(filename string) error{

	/*
		如果filename路径不存在，就新建它
	*/	
	var exist func(string) bool = func (s string) bool {
		_, err := os.Stat(s)
		return err == nil || os.IsExist(err)
	}

	for i:=0;true;{
		a := strings.Index(filename[i:],"/")
		if a == -1 {break}
		if a == 0 {a = 1}//bug fix 当绝对路径时开头的/导致问题
		i=i+a+1
		if !exist(filename[:i-1]) {
			err := os.Mkdir(filename[:i-1], os.ModePerm)
			if err != nil {return err}
		}
	}
	
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fd,err:=os.Create(filename)
		if err != nil {
			return err
		}else{
			fd.Close()
		}
	}
	return nil
}

func FileMove(src,trg string) error {
	return Ppart.FileMove(src,trg)
}

// func main(){
// 	var u File 
// 	u.File="a.txt"
// 	u.Write=false
// 	u.Loc=0
// 	u.ReadNum=2
// 	u.Context="ad"
// 	fmt.Println(FileWR(u))
// }
