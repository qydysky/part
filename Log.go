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
    levelName []string
    base []interface{}
    baset int

    channelN chan int
    channel chan interface{}
    wantLog chan bool
    blog chan int

    sync.Mutex

    started bool
    logging bool
    pause bool
    fileonly bool

    sleep sync.Mutex
}

const (
    defaultlevelName0 = "DEBUG"
    defaultlevelName1 = "INFO"
    defaultlevelName2 = "WARNING"
    defaultlevelName3 = "ERROR"
    defaultchannelMax = 1e4
)

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
        O.channelMax = defaultchannelMax
    }
    if len(O.levelName) == 0 {
        O.levelName = []string{defaultlevelName0,defaultlevelName1,defaultlevelName2,defaultlevelName3}
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

        var msg []interface{}
        if O.baset != 0 {
            if O.baset > 0 {O.baset -= 1}
            msg = append(msg, O.base)
        }
        msg = append(msg, <- O.channel)
        
        if channelN >= 0 && channelN < O.level {continue}

        var showObj []io.Writer
        if file != nil {showObj = append(showObj, file)}
        if file == nil || !O.fileonly {showObj = append(showObj, os.Stdout)}
        
        switch channelN {
        case blockErr:
            tmp = channelN
        case nofileErr, waitErr:
            return channelN
        case 0:
            log.New(io.MultiWriter(showObj...),
            O.levelName[0] + ": ",
            log.Ldate|log.Ltime).Println(msg...)
        case 1:
            log.New(io.MultiWriter(showObj...),
            O.levelName[1] + ": ",
            log.Ldate|log.Ltime).Println(msg...)
        case 2:
            log.New(io.MultiWriter(showObj...),
            O.levelName[2] + ": ",
            log.Ldate|log.Ltime).Println(msg...)
        case 3:
            log.New(io.MultiWriter(showObj...),
            O.levelName[3] + ": ",
            log.Ldate|log.Ltime).Println(msg...)
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

//LevelName 设置日志等级名
func (I *logl) LevelName(s []string) (O *logl) {
    O=I
    if O.started {O.E("LevelName() must be called before New()");return}
    if len(s) != 4 {O.E("len(LevelName) != 4");return}
    O.levelName = s
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
    O.blog <- waitErr
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
    O.blog <- blockErr
    return
}

//MTimeout 阻塞超时毫秒数
func (I *logl) MTimeout(t int) (O *logl) {
    O=I
    go func(O *logl){
        Sys().MTimeoutf(t);
        if len(O.blog) == 0 {
            O.blog <- blockErr
            O.blog <- waitErr
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

//Fileonly 不输出到屏幕
func (I *logl) Fileonly(s bool) (O *logl) {
    O=I
    if O.fileName == "" {O.E("No set filename yet! ignore!");return}
    O.Block().fileonly = s
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
//Base 追加到后续输出,t可追加次数(负数为不计数,0为取消)
func (I *logl) Base(t int, i ...interface{}) (O *logl) {
    O=I
    O.baset = t
    O.base = i
    return
}
func (I *logl) T(i ...interface{}) (O *logl) {
    O=I
    if !O.started {log.New(io.MultiWriter(os.Stdout),defaultlevelName0 + ": ",log.Ldate|log.Ltime).Println(i...);return}
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
    if !O.started {log.New(io.MultiWriter(os.Stdout),defaultlevelName1 + ": ",log.Ldate|log.Ltime).Println(i...);return}
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
    if !O.started {log.New(io.MultiWriter(os.Stdout),defaultlevelName2 + ": ",log.Ldate|log.Ltime).Println(i...);return}
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
    if !O.started {log.New(io.MultiWriter(os.Stdout),defaultlevelName3 + ": ",log.Ldate|log.Ltime).Println(i...);return}
    O.sleep.Lock()
    O.checkDrop()
    O.channelN <- 3
    O.channel <- i
    O.sleep.Unlock()
    if !O.logging && !O.pause {O.wantLog <- true}
    return
}