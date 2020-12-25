package part

import (
	"io"
	"time"
	"os"
    "log"

    p "github.com/qydysky/part"
    m "github.com/qydysky/part/msgq"
)

var (
    On = struct{}{}
)

type Log_interface struct {
    MQ *m.Msgq
    Config
}

type Config struct {
    File_string string
    Prefix_string map[string]struct{}
    Base_string []interface{}
}

type Msg_item struct {
    Prefix string
    Show_obj []io.Writer
    Msg_obj []interface{}
}

//New 初始化
func New(c Config) (o *Log_interface) {

    o = new(Log_interface)
    o.Prefix_string = make(map[string]struct{})

    //设置log等级字符串
    for k,_ := range c.Prefix_string {
        o.Prefix_string[k] = On
    } 

    o.MQ = m.New(10)

    o.MQ.Pull_tag(map[string]func(interface{})(bool){
        `block`:func(data interface{})(bool){
            if v,ok := data.(chan struct{});ok{close(v)}
            return false
        },
        `L`:func(data interface{})(bool){
            log.New(io.MultiWriter(data.(Msg_item).Show_obj...),
            data.(Msg_item).Prefix,
            log.Ldate|log.Ltime).Println(data.(Msg_item).Msg_obj...)
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
func (I *Log_interface) Level(log map[string]struct{}) (O *Log_interface) {
    O = copy(I)
    for k,_ := range O.Prefix_string {
        if _,ok := log[k];!ok{delete(O.Prefix_string,k)}
    }
    return
}


//Open 日志输出至文件
func (I *Log_interface) Log_to_file(fileP string) (O *Log_interface) {
    O=I
    O.File_string = fileP
    if fileP == `` {return}
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
func (I *Log_interface) L(prefix string, i ...interface{}) (O *Log_interface) {
    O=I
    if _,ok := O.Prefix_string[prefix];!ok{return}
    var showObj = []io.Writer{os.Stdout}

    if O.File_string != `` {
        file, err := os.OpenFile(O.File_string, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {O.L("Error: ","Failed to open log file:", err)}else{showObj = append(showObj, file)}
    }
    O.MQ.Push_tag(`L`,Msg_item{
        Prefix:prefix,
        Show_obj:showObj,
        Msg_obj:append(O.Base_string, i),
    })
    return
}