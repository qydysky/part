package part

import (
	"io"
	"os"
    "log"
    "sync"
)

type logl struct {
    fileName string
    channelMax int
    level int
    channelN chan int
    channel chan interface{}
    wantLog chan bool
    blog chan int
    sync.Mutex
    started bool
    logging bool
    pause bool

    sleep sync.Mutex
}

const (
    blockErr = -3
    waitErr = -2
    nofileErr = -1
)

func Logf() (*logl) {
	return new(logl)
}

//New 初始化
func (I *logl) New() (O *logl) {
    O=I
    if O.channelMax == 0 {
        O.channelMax = 1e4
    }
    O.channelN = make(chan int,O.channelMax)
    O.channel = make(chan interface{},O.channelMax)
    O.wantLog = make(chan bool,10)
    O.blog = make(chan int,1)
    O.started = true

    go func(){
        for {
            
            O.logging = false
            for ;len(O.channelN) == 0;<- O.wantLog {}
            O.logging = true

            var (
                file *os.File
                err error
            )

            fileName := O.fileName
            if fileName != "" {
                O.Lock()
                File().NewPath(fileName)
                file, err = os.OpenFile(fileName,
                    os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
                if err != nil {
                    O.E("Failed to open log file:", err)
                }
            }

            tmpsign := O.logf(file)

            if fileName != "" {file.Close();O.Unlock()}
            
            switch tmpsign {
            case nofileErr:O.Close()
            case blockErr:O.CloseBlock()
            case waitErr:O.CloseWait()
            default:;
            }

        }
    }()
    return
}

func (O *logl) logf(file *os.File) (int) {
    var tmp int
    for len(O.channelN) != 0 {
        channelN := <- O.channelN
        channel := <- O.channel
        if channelN >= 0 && channelN < O.level {continue}
        switch channelN {
        case blockErr:
            tmp = channelN
        case nofileErr, waitErr:
            return channelN
        case 0:
            log.New(io.MultiWriter(os.Stdout, file),
            "TRACE: "+O.fileName+" ",
            log.Ldate|log.Ltime).Println(channel)
        case 1:
            log.New(io.MultiWriter(os.Stdout, file),
            "INFO: "+O.fileName+" ",
            log.Ldate|log.Ltime).Println(channel)
        case 2:
            log.New(io.MultiWriter(os.Stdout, file),
            "WARNING: "+O.fileName+" ",
            log.Ldate|log.Ltime).Println(channel)
        case 3:
            log.New(io.MultiWriter(os.Stdout, file),
            "ERROR: "+O.fileName+" ",
            log.Ldate|log.Ltime).Println(channel)
        default:;
        }
    }
    return tmp
}

//Level 设置之后日志等级
func (I *logl) Level(l int) (O *logl) {
    O=I
    if l < 0 {l = 0}
    if l > 3 {l = 4}
    O.Block().level = l
    return
}

//BufSize 设置日志缓冲数量
func (I *logl) BufSize(s int) (O *logl) {
    O=I
    if O.started {O.E("BufSize() must be called before New()");return}
    if s < 1 {s = 1}
    O.channelMax = s
    return
}
//Len 获取日志缓冲数量
func (O *logl) Len() (int) {
    return len(O.channelN)
}

//Open 立即将日志输出至文件
func (I *logl) Open(fileP string) (O *logl) {
    O=I
    O.fileName = fileP
    return
}

//Close 立即停止文件日志输出
func (I *logl) Close() (O *logl) {
    O=I
    O.Open("")
    return
}

//NoFile 之后的日志不再输出至文件
func (I *logl) NoFile() (O *logl) {
    O=I
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- nofileErr
    O.channel <- []interface{}{}
    O.sleep.Unlock()
    if !O.logging {O.wantLog <- true}
    return
}

//Wait 阻塞直至(等待)日志到来
func (I *logl) Wait() (O *logl) {
    O=I
    for ;len(O.blog) != 0;<-O.blog {}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- waitErr
    O.channel <- []interface{}{}
    O.sleep.Unlock()
    for <-O.blog != waitErr {}
    return
}

//CloseWait 停止等待
func (I *logl) CloseWait() (O *logl) {
    O=I
    if len(O.blog) != 0 {O.E("Other Close-Function has been called! Cancel!");return}
    O.blog <- -3
    return
}

//Block 阻塞直到本轮日志输出完毕
func (I *logl) Block() (O *logl) {
    O=I
    for ;len(O.blog) != 0;<-O.blog {}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- blockErr
    O.channel <- []interface{}{}
    O.sleep.Unlock()
    if !O.logging {O.wantLog <- true}
    for <-O.blog != blockErr {}
    return
}

//CloseBlock 停止阻塞
func (I *logl) CloseBlock() (O *logl) {
    O=I
    if len(O.blog) != 0 {O.E("Other Close-Function has been called! Cancel!");return}
    O.blog <- -2
    return
}

//MTimeout 阻塞超时毫秒数
func (I *logl) MTimeout(t int) (O *logl) {
    O=I
    go func(O *logl){
        Sys().MTimeoutf(t);
        if len(O.blog) == 0 {
            O.blog <- -3
            O.blog <- -2
        }
    }(O)
    return
}

//Pause 之后暂停输出，仅接受日志
func (I *logl) Pause(s bool) (O *logl) {
    O=I
    O.Block().pause = s
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}

func (I *logl) checkDrop() (O *logl) {
    O=I
    if O.pause && len(O.channelN) == O.channelMax {<- O.channelN;<- O.channel}
    return
}

func (I *logl) clearup() (O *logl) {
    O=I
    for ;len(O.channelN) != 0;<-O.channelN {}
    for ;len(O.channel) != 0;<-O.channel {}
    return
}

//组合使用
func (I *logl) BC() (O *logl) {
    O=I
    O.Block().Close()
    return
}

func (I *logl) NC() (O *logl) {
    O=I
    O.NoFile().Close()
    return
}


//日志等级
func (I *logl) T(i ...interface{}) (O *logl) {
    O=I
    if !O.started {log.New(io.MultiWriter(os.Stdout),"TRACE: ",log.Ldate|log.Ltime).Println(i...);return}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- 0
    O.channel <- i
    O.sleep.Unlock()
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}
func (I *logl) I(i ...interface{}) (O *logl) {
    O=I
    if !O.started {log.New(io.MultiWriter(os.Stdout),"INFO: ",log.Ldate|log.Ltime).Println(i...);return}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- 1
    O.channel <- i
    O.sleep.Unlock()
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}
func (I *logl) W(i ...interface{}) (O *logl) {
    O=I
    if !O.started {log.New(io.MultiWriter(os.Stdout),"WARNING: ",log.Ldate|log.Ltime).Println(i...);return}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- 2
    O.channel <- i
    O.sleep.Unlock()
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}
func (I *logl) E(i ...interface{}) (O *logl) {
    O=I
    if !O.started {log.New(io.MultiWriter(os.Stdout),"ERROR: ",log.Ldate|log.Ltime).Println(i...);return}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- 3
    O.channel <- i
    O.sleep.Unlock()
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}