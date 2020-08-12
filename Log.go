package part

import (
	"io"
	"os"
    "log"
    "syscall"
)

type logl struct {
    fileName string
    channelN chan int
    channel chan interface{}
    wantLog chan bool
    waitclose chan bool
    tracef   *log.Logger // 记录所有日志
    infof    *log.Logger // 重要的信息
    warningf *log.Logger // 需要注意的信息
    errorf   *log.Logger // 非常严重的问题
}

func Logf() (*logl) {
	return &logl{}
}

func (l *logl) New(fileP string) {
    l.wantLog = make(chan bool,2)
    l.waitclose = make(chan bool)
    l.channelN = make(chan int,200)
    l.channel = make(chan interface{},200)

    l.fileName = fileP
        
    go func(){
        for {
            <- l.wantLog

            if len(l.channel) == 0 {continue}

            fileName := l.fileName
            if fileName == "" {
                for len(l.channel) != 0 {
                    i := <- l.channelN
                    switch i {
                    case 0:
                        log.Println("TRACE:",<- l.channel)
                    case 1:
                        log.Println("INFO:",<- l.channel)
                    case 2:
                        log.Println("WARNING:",<- l.channel)
                    case 3:
                        log.Println("ERROR:",<- l.channel)
                    }
                }
                continue
            }

            File().NewPath(fileName)
            file, err := os.OpenFile(fileName,
                os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
            if err != nil {
                l.E("Failed to open log file:", err)
                continue
            }

            err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
            if err != nil {
                l.E("cannot flock", fileName, err)
                file.Close()
                continue
            }
            
            l.tracef = log.New(io.MultiWriter(file, os.Stdout),
            "TRACE: "+fileName+" ",
            log.Ldate|log.Ltime)
        
            l.infof = log.New(io.MultiWriter(file, os.Stdout),
                "INFO: "+fileName+" ",
                log.Ldate|log.Ltime)
        
            l.warningf = log.New(io.MultiWriter(file, os.Stdout),
                "WARNING: "+fileName+" ",
                log.Ldate|log.Ltime)
        
            l.errorf = log.New(io.MultiWriter(file, os.Stderr),
                "ERROR: "+fileName+" ",
                log.Ldate|log.Ltime)
        
            l.tracef.Println("Log chan",len(l.channel))
        
            for len(l.channelN) != 0 {
                i := <- l.channelN
                switch i {
                case -1:
                    l.fileName = ""
                    l.waitclose <- true
                case 0:
                    l.tracef.Println(<- l.channel)
                case 1:
	                l.infof.Println(<- l.channel)
                case 2:
                    l.warningf.Println(<- l.channel)
                case 3:
                    l.errorf.Println(<- l.channel)
                }
            }

            err = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
            if err != nil {
                log.Fatalln("cannot Unflock", fileName, err)
            }

            file.Close()

            <- l.wantLog
        }
    }()
}

func (l *logl) Close(){
    l.fileName = ""
}

func (l *logl) WClose(){
    l.channelN <- -1
    if len(l.wantLog) ==0 {l.wantLog <- true;l.wantLog <- true}
    <- l.waitclose
}

func (l *logl) T(i ...interface{}){
    if l.fileName == "" {log.Println("TRACE:", i);return}
    l.channelN <- 0
    l.channel <- i
    if len(l.wantLog) ==0 {l.wantLog <- true;l.wantLog <- true}
}
func (l *logl) I(i ...interface{}){
    if l.fileName == "" {log.Println("INFO:", i);return}
    l.channelN <- 1
    l.channel <- i
    if len(l.wantLog) ==0 {l.wantLog <- true;l.wantLog <- true}
}
func (l *logl) W(i ...interface{}){
    if l.fileName == "" {log.Println("WARNING:", i);return}
    l.channelN <- 2
    l.channel <- i
    if len(l.wantLog) ==0 {l.wantLog <- true;l.wantLog <- true}
}
func (l *logl) E(i ...interface{}){
    if l.fileName == "" {log.Println("ERROR:", i);return}
    l.channelN <- 3
    l.channel <- i
    if len(l.wantLog) ==0 {l.wantLog <- true;l.wantLog <- true}
}