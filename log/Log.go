package part

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	pctx "github.com/qydysky/part/ctx"
	f "github.com/qydysky/part/file"
	m "github.com/qydysky/part/msgq"
	psql "github.com/qydysky/part/sql"
)

var (
	On = struct{}{}
)

type Log_interface struct {
	MQ *m.MsgType[Msg_item]
	Config
}

type Config struct {
	To       time.Duration
	File     string
	DBConn   *sql.DB
	DBConnTo time.Duration

	// $1:Prefix $2:Base $2:Msgs
	DBInsert string
	Stdout   bool

	Prefix_string map[string]struct{}
	Base_string   []any
}

type Msg_item struct {
	Prefix string
	Msgs   []any
	Config
}

// New 初始化
func New(c Config) (o *Log_interface) {

	o = &Log_interface{
		Config: c,
	}
	if c.File != `` {
		f.New(c.File, 0, true).Create()
	}
	if o.To != 0 {
		o.MQ = m.NewType[Msg_item](o.To)
	} else {
		o.MQ = m.NewType[Msg_item]()
	}

	o.MQ.Pull_tag_only(`L`, func(msg Msg_item) bool {
		var showObj = []io.Writer{}
		if msg.Stdout {
			showObj = append(showObj, os.Stdout)
		}
		if msg.File != `` {
			file, err := os.OpenFile(msg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				showObj = append(showObj, file)
				defer file.Close()
			} else {
				log.Println(err)
			}
		}
		if msg.DBConn != nil && msg.DBInsert != `` {
			var sqlTx *psql.SqlTx[any]
			if o.DBConnTo == 0 {
				sqlTx = psql.BeginTx[any](msg.DBConn, context.Background())
			} else {
				sqlTx = psql.BeginTx[any](msg.DBConn, pctx.GenTOCtx(o.DBConnTo))
			}
			sqlTx.SimpleDo(
				msg.DBInsert,
				strings.TrimSpace(msg.Prefix),
				strings.TrimSpace(fmt.Sprintln(msg.Base_string...)),
				strings.TrimSpace(fmt.Sprintln(msg.Msgs...)))
			if _, err := sqlTx.Fin(); err != nil {
				log.Println(err)
			}
		}
		log.New(io.MultiWriter(showObj...),
			msg.Prefix,
			log.Ldate|log.Ltime).Println(append(msg.Base_string, msg.Msgs...))
		return false
	})
	//启动阻塞
	o.MQ.PushLock_tag(`block`, Msg_item{})
	return
}

func Copy(i *Log_interface) (o *Log_interface) {
	o = &Log_interface{
		Config: (*i).Config,
		MQ:     (*i).MQ,
	}
	return
}

// Level 设置之后日志等级
func (I *Log_interface) Level(log map[string]struct{}) (O *Log_interface) {
	O = Copy(I)
	for k := range O.Prefix_string {
		if _, ok := log[k]; !ok {
			delete(O.Prefix_string, k)
		}
	}
	return
}

// Open 日志不显示
func (I *Log_interface) Log_show_control(show bool) (O *Log_interface) {
	O = Copy(I)
	O.Stdout = show
	return
}

func (I *Log_interface) LShow(show bool) (O *Log_interface) {
	return I.Log_show_control(show)
}

// Open 日志输出至文件
func (I *Log_interface) Log_to_file(fileP string) (O *Log_interface) {
	O = I
	if O.File != `` && fileP != `` {
		O.File = fileP
		f.New(O.File, 0, true).Create()
	} else {
		O.File = ``
	}
	return
}

// Open 日志输出至DB
func (I *Log_interface) LDB(db *sql.DB, insert string, to ...time.Duration) (O *Log_interface) {
	O = I
	if db != nil && insert != `` {
		O.DBConn = db
		O.DBInsert = insert
		if len(to) > 0 {
			O.DBConnTo = to[0]
		}
	} else {
		O.DBConn = nil
		O.DBInsert = ``
		O.DBConnTo = 0
	}
	return
}

func (I *Log_interface) LFile(fileP string) (O *Log_interface) {
	return I.Log_to_file(fileP)
}

func (I *Log_interface) Close() {
	I.MQ.ClearAll()
	if I.DBConn != nil {
		(*I.DBConn).Close()
	}
}

// 日志等级
// Base 追加到后续输出
func (I *Log_interface) Base(i ...any) (O *Log_interface) {
	O = Copy(I)
	O.Base_string = i
	return
}
func (I *Log_interface) Base_add(i ...any) (O *Log_interface) {
	O = Copy(I)
	O.Base_string = append(O.Base_string, i...)
	return
}
func (I *Log_interface) BaseAdd(i ...any) (O *Log_interface) {
	return I.Base_add(i...)
}
func (I *Log_interface) L(prefix string, i ...any) (O *Log_interface) {
	O = I
	if _, ok := O.Prefix_string[prefix]; !ok {
		return
	}

	O.MQ.Push_tag(`L`, Msg_item{
		Prefix: prefix,
		Msgs:   i,
		Config: O.Config,
	})
	return
}

func (I *Log_interface) Debug(msg string, args ...any) {
	I.L(msg, args...)
}
func (I *Log_interface) Info(msg string, args ...any) {
	I.L(msg, args...)
}
func (I *Log_interface) Warn(msg string, args ...any) {
	I.L(msg, args...)
}
func (I *Log_interface) Error(msg string, args ...any) {
	I.L(msg, args...)
}
