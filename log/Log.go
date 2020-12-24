package part

import (
	"io"
	"time"
	"os"
    "log"

    p "github.com/qydysky/part"
    m "github.com/qydysky/part/msgq"
)

type Log_interface struct {
    MQ *m.Msgq
    Config
}

type Config struct {
    File_string string
    Level_string [4]string
    Base_string []interface{}
}

type msg_item struct {
    Show_obj []io.Writer
    Msg_obj []interface{}
}

//New 初始化
func New(c Config) (o *Log_interface) {

    o = new(Log_interface)
    
    //设置log等级字符串
    for k,v := range c.Level_string {
        if v == `` {continue}
        o.Level_string[k] = v
    } 

    o.MQ = m.New(10)

    o.MQ.Pull_tag(map[string]func(interface{})(bool){
        `block`:func(data interface{})(bool){
            if v,ok := data.(chan struct{});ok{close(v)}
            return false
        },
        `T`:func(data interface{})(bool){
            log.New(io.MultiWriter(data.(msg_item).Show_obj...),
            o.Level_string[0] + ": ",
            log.Ldate|log.Ltime).Println(data.(msg_item).Msg_obj...)
            return false
        },
        `I`:func(data interface{})(bool){
            log.New(io.MultiWriter(data.(msg_item).Show_obj...),
            o.Level_string[1] + ": ",
            log.Ldate|log.Ltime).Println(data.(msg_item).Msg_obj...)
            return false
        },
        `W`:func(data interface{})(bool){
            log.New(io.MultiWriter(data.(msg_item).Show_obj...),
            o.Level_string[2] + ": ",
            log.Ldate|log.Ltime).Println(data.(msg_item).Msg_obj...)
            return false
        },
        `E`:func(data interface{})(bool){
            log.New(io.MultiWriter(data.(msg_item).Show_obj...),
            o.Level_string[3] + ": ",
            log.Ldate|log.Ltime).Println(data.(msg_item).Msg_obj...)
            return false
        },
    })
    return
}

//
func copy(i *Log_interface)(o *Log_interface){
    t := *i
    o = &t
    return
}

//Level 设置之后日志等级
func (I *Log_interface) Level(l int) (O *Log_interface) {
    O = copy(I)
    for k,_ := range O.Level_string {
        if k < l {O.Level_string[k] = ``}
    }
    return
}


//Open 日志输出至文件
func (I *Log_interface) Log_to_file(fileP string) (O *Log_interface) {
    O=I
    O.File_string = fileP
    p.File().NewPath(fileP)
    return
}

//Block 阻塞直到本轮日志输出完毕
func (I *Log_interface) Block(timeout int) (O *Log_interface) {
    O=I
    b := make(chan struct{})
    O.MQ.Push_tag(`block`,b)
    select {
    case <-b:
    case <-time.After(time.Duration(timeout)*time.Millisecond):
    }
    return
}

//日志等级
//Base 追加到后续输出
func (I *Log_interface) Base(i ...interface{}) (O *Log_interface) {
    O=copy(I)
    O.Base_string = i
    return
}
func (I *Log_interface) Base_add(i ...interface{}) (O *Log_interface) {
    O=copy(I)
    O.Base_string = append(O.Base_string, i...)
    return
}
func (I *Log_interface) T(i ...interface{}) (O *Log_interface) {
    O=I
    if O.Level_string[0] == `` {return}
    var showObj = []io.Writer{os.Stdout}

    if O.File_string != `` {
        file, err := os.OpenFile(O.File_string, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {O.E("Failed to open log file:", err)}else{showObj = append(showObj, file)}
    }
    O.MQ.Push_tag(`T`,msg_item{
        Show_obj:showObj,
        Msg_obj:append(O.Base_string, i),
    })
    return
}
func (I *Log_interface) I(i ...interface{}) (O *Log_interface) {
    O=I
    if O.Level_string[1] == `` {return}
    var showObj = []io.Writer{os.Stdout}

    if O.File_string != `` {
        file, err := os.OpenFile(O.File_string, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {O.E("Failed to open log file:", err)}else{showObj = append(showObj, file)}
    }
    O.MQ.Push_tag(`I`,msg_item{
        Show_obj:showObj,
        Msg_obj:append(O.Base_string, i),
    })
    return
}
func (I *Log_interface) W(i ...interface{}) (O *Log_interface) {
    O=I
    if O.Level_string[2] == `` {return}
    var showObj = []io.Writer{os.Stdout}

    if O.File_string != `` {
        file, err := os.OpenFile(O.File_string, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {O.E("Failed to open log file:", err)}else{showObj = append(showObj, file)}
    }
    O.MQ.Push_tag(`W`,msg_item{
        Show_obj:showObj,
        Msg_obj:append(O.Base_string, i),
    })
    return
}
func (I *Log_interface) E(i ...interface{}) (O *Log_interface) {
    O=I
    if O.Level_string[3] == `` {return}
    var showObj = []io.Writer{os.Stdout}

    if O.File_string != `` {
        file, err := os.OpenFile(O.File_string, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {O.E("Failed to open log file:", err)}else{showObj = append(showObj, file)}
    }
    O.MQ.Push_tag(`E`,msg_item{
        Show_obj:showObj,
        Msg_obj:append(O.Base_string, i),
    })
    return
}