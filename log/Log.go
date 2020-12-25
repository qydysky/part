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
    File string
    Prefix_string map[string]struct{}
    Base_string []interface{}
}

type Msg_item struct {
    Prefix string
    Msg_obj []interface{}
}

//New 初始化
func New(c Config) (o *Log_interface) {

    o = new(Log_interface)

    //设置
    o.Config = c
    
    if c.File != `` {p.File().NewPath(c.File)}

    o.MQ = m.New(10)
    o.MQ.Pull_tag(map[string]func(interface{})(bool){
        `block`:func(data interface{})(bool){
            if v,ok := data.(chan struct{});ok{close(v)}
            return false
        },
        `L`:func(data interface{})(bool){
            var showObj = []io.Writer{os.Stdout}
            if o.File != `` {
                file, err := os.OpenFile(o.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
                if err == nil {
                    showObj = append(showObj, file)
                    defer file.Close()
                }else{log.Println(err)}
            }
            log.New(io.MultiWriter(showObj...),
            data.(Msg_item).Prefix,
            log.Ldate|log.Ltime).Println(data.(Msg_item).Msg_obj...)
            return false
        },
    })
    {//启动阻塞
        b := make(chan struct{})
        i := true
        for i {
            select {
            case <-b:i = false
            case <-time.After(time.Duration(10)*time.Millisecond):o.MQ.Push_tag(`block`,b)
            }
        }
    }    
    return
}

//
func copy(i *Log_interface)(o *Log_interface){
    o = New((*i).Config)
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
    O=copy(I)
    O.File = fileP
    if O.File != `` {p.File().NewPath(O.File)}
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

    O.MQ.Push_tag(`L`,Msg_item{
        Prefix:prefix,
        Msg_obj:append(O.Base_string, i),
    })
    return
}