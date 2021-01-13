package part

import (
	"io"
	"time"
	"os"
    "log"

    p "github.com/qydysky/part"
    m "github.com/qydysky/part/msgq"
    s "github.com/qydysky/part/signal"
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
    Stdout bool
    Prefix_string map[string]struct{}
    Base_string []interface{}
}

type Msg_item struct {
    Prefix string
    Msg_obj []interface{}
    Config
}

//New 初始化
func New(c Config) (o *Log_interface) {

    o = &Log_interface{
        Config:c,
    }
    if c.File != `` {p.File().NewPath(c.File)}

    o.MQ = m.New(100)
    o.MQ.Pull_tag(map[string]func(interface{})(bool){
        `block`:func(data interface{})(bool){
            if v,ok := data.(*s.Signal);ok{v.Done()}
            return false
        },
        `L`:func(data interface{})(bool){
            msg := data.(Msg_item)
            var showObj = []io.Writer{}
            if msg.Stdout {showObj = append(showObj, os.Stdout)} 
            if msg.File != `` {
                file, err := os.OpenFile(msg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
                if err == nil {
                    showObj = append(showObj, file)
                    defer file.Close()
                }else{log.Println(err)}
            }
            log.New(io.MultiWriter(showObj...),
            msg.Prefix,
            log.Ldate|log.Ltime).Println(msg.Msg_obj...)
            return false
        },
    })
    {//启动阻塞
        b := s.Init()
        for b.Islive() {
            o.MQ.Push_tag(`block`,b)
            time.Sleep(time.Duration(20)*time.Millisecond)
        }
    }    
    return
}

//
func Copy(i *Log_interface)(o *Log_interface){
    o = &Log_interface{
        Config:(*i).Config,
        MQ:(*i).MQ,
    }
    {//启动阻塞
        b := s.Init()
        for b.Islive() {
            o.MQ.Push_tag(`block`,b)
            time.Sleep(time.Duration(20)*time.Millisecond)
        }
    }
    return
}

//Level 设置之后日志等级
func (I *Log_interface) Level(log map[string]struct{}) (O *Log_interface) {
    O = Copy(I)
    for k,_ := range O.Prefix_string {
        if _,ok := log[k];!ok{delete(O.Prefix_string,k)}
    }
    return
}

//Open 日志不显示
func (I *Log_interface) Log_show_control(show bool) (O *Log_interface) {
    O = Copy(I)
    //
    O.Block(100)
    O.Stdout = show
    return
}

//Open 日志输出至文件
func (I *Log_interface) Log_to_file(fileP string) (O *Log_interface) {
    O=I
    //
    O.Block(100)
    O.File = fileP
    if O.File != `` {p.File().NewPath(O.File)}
    return
}

//Block 阻塞直到本轮日志输出完毕
func (I *Log_interface) Block(timeout int) (O *Log_interface) {
    O=I
    b := s.Init()
    O.MQ.Push_tag(`block`,b)
    b.Wait()
    return
}

//日志等级
//Base 追加到后续输出
func (I *Log_interface) Base(i ...interface{}) (O *Log_interface) {
    O=Copy(I)
    O.Base_string = i
    return
}
func (I *Log_interface) Base_add(i ...interface{}) (O *Log_interface) {
    O=Copy(I)
    O.Base_string = append(O.Base_string, i...)
    return
}
func (I *Log_interface) L(prefix string, i ...interface{}) (O *Log_interface) {
    O=I
    if _,ok := O.Prefix_string[prefix];!ok{return}

    O.MQ.Push_tag(`L`,Msg_item{
        Prefix:prefix,
        Msg_obj:append(O.Base_string, i),
        Config:O.Config,
    })
    return
}