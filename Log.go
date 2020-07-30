package part

import (
	"io"
	"os"
	"log"
)

type logl struct {
    file string
    tracef   *log.Logger // 记录所有日志
    infof    *log.Logger // 重要的信息
    warningf *log.Logger // 需要注意的信息
    errorf   *log.Logger // 非常严重的问题
}

func Logf() (*logl) {
	return &logl{}
}

func (l *logl) New(fileP string) {

    File().NewPath(fileP)
    
    file, err := os.OpenFile(fileP,
        os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalln("Failed to open error log file:", err)
    }

    l.tracef = log.New(io.MultiWriter(file, os.Stdout),
        "TRACE: "+fileP+" ",
        log.Ldate|log.Ltime)

    l.infof = log.New(io.MultiWriter(file, os.Stdout),
        "INFO: "+fileP+" ",
        log.Ldate|log.Ltime)

    l.warningf = log.New(io.MultiWriter(file, os.Stdout),
        "WARNING: "+fileP+" ",
        log.Ldate|log.Ltime)

    l.errorf = log.New(io.MultiWriter(file, os.Stderr),
        "ERROR: "+fileP+" ",
		log.Ldate|log.Ltime)
		
    l.file = fileP
}

func (l *logl) T(i ...interface{}){
	if l.file == "" {log.Println("TRACE:",i);return}
	l.tracef.Println(i...)
}
func (l *logl) I(i ...interface{}){
	if l.file == "" {log.Println("INFO:",i);return}
	l.infof.Println(i...)
}
func (l *logl) W(i ...interface{}){
	if l.file == "" {log.Println("WARNING:",i);return}
	l.warningf.Println(i...)
}
func (l *logl) E(i ...interface{}){
	if l.file == "" {log.Println("ERROR:",i);return}
	l.errorf.Println(i...)
}