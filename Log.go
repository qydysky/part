package part

import (
	"io"
	"os"
	"log"
)

type logl struct {}

func Logf() (*logl) {
	return &logl{}
}

var (
	isinit bool
    tracef   *log.Logger // 记录所有日志
    infof    *log.Logger // 重要的信息
    warningf *log.Logger // 需要注意的信息
    errorf   *log.Logger // 非常严重的问题
)

func (*logl) New(fileP string) {

    File().NewPath(fileP)
    
    file, err := os.OpenFile(fileP,
        os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalln("Failed to open error log file:", err)
    }

    tracef = log.New(io.MultiWriter(file, os.Stdout),
        "TRACE: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    infof = log.New(io.MultiWriter(file, os.Stdout),
        "INFO: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    warningf = log.New(io.MultiWriter(file, os.Stdout),
        "WARNING: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    errorf = log.New(io.MultiWriter(file, os.Stderr),
        "ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
		
	isinit = true
}

func (*logl) T(l ...string){
	if !isinit {log.Println("TRACE:",l);return}
	tracef.Println(l)
}
func (*logl) I(l ...string){
	if !isinit {log.Println("INFO:",l);return}
	infof.Println(l)
}
func (*logl) W(l ...string){
	if !isinit {log.Println("WARNING:",l);return}
	warningf.Println(l)
}
func (*logl) E(l ...string){
	if !isinit {log.Println("ERROR:",l);return}
	errorf.Println(l)
}