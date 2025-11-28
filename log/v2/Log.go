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
	pool "github.com/qydysky/part/pool"
	psql "github.com/qydysky/part/sql"
	psys "github.com/qydysky/part/sys"
)

type LogI struct {
	MQ *m.MsgType[*MsgItem]
	Config
}

type Config struct {
	To time.Duration

	File string

	// type LogDb struct {
	// 	Date   string
	// 	Unix   int64
	// 	Prefix string
	// 	Base   string
	// 	Msgs   string
	// }
	DBInsert string
	DBHolder psql.ReplaceF
	DBConn   *sql.DB
	dbPool   *psql.TxPool[any]
	dbInsert *psql.SqlFunc[any]

	NoStdout bool

	PrefixS map[Level]string
	BaseS   []any

	logger *log.Logger
}

func (o *Config) reloadLogger() {
	// logger
	var showObj = []io.Writer{}
	if !o.NoStdout {
		showObj = append(showObj, os.Stdout)
	}
	if o.File != `` {
		file, err := os.OpenFile(o.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			showObj = append(showObj, file)
			defer file.Close()
		} else {
			log.Println(err)
		}
	}
	o.logger = log.New(io.MultiWriter(showObj...), "", log.Ldate|log.Ltime)
}

type MsgItem struct {
	prefix Level
	format string
	msgs   []any
	Config
}

type LogDb struct {
	Date   string
	Unix   int64
	Prefix string
	Base   string
	Msgs   string
}

type Level int

const (
	T Level = iota
	I
	W
	E
)

// New 初始化
func New(c Config) (o *LogI) {
	o = &LogI{
		Config: c,
	}
	if c.File != `` {
		f.New(c.File, 0, true).Create()
	}
	if o.To != 0 {
		o.MQ = m.NewType[*MsgItem](o.To)
	} else {
		o.MQ = m.NewType[*MsgItem]()
	}
	if o.DBConn != nil && o.DBInsert != `` && o.DBHolder != nil {
		o.dbPool = psql.NewTxPool[any](o.DBConn)
		o.dbInsert = &psql.SqlFunc[any]{Sql: o.DBInsert}
	}
	if o.PrefixS == nil {
		o.PrefixS = map[Level]string{T: "T:", I: "I:", W: "W:", E: "E:"}
	}

	o.reloadLogger()

	formatPool := pool.NewPoolBlocks[byte]()
	valPool := pool.NewPoolBlocks[any]()

	o.MQ.Pull_tag_only(`L`, func(msg *MsgItem) bool {
		if msg.dbPool != nil {
			var sqlTx *psql.SqlTx[any]
			if o.To == 0 {
				sqlTx = msg.dbPool.BeginTx(context.Background())
			} else {
				sqlTx = msg.dbPool.BeginTx(pctx.GenTOCtx(o.To))
			}
			sqlTx.DoPlaceHolder(msg.dbInsert, &LogDb{
				Date:   time.Now().Format(time.DateTime),
				Unix:   time.Now().Unix(),
				Prefix: strings.TrimSpace(msg.PrefixS[msg.prefix]),
				Base:   strings.TrimSpace(fmt.Sprintln(msg.BaseS...)),
				Msgs:   strings.TrimSpace(fmt.Sprintln(msg.msgs...)),
			}, msg.DBHolder)
			if _, err := sqlTx.Fin(); err != nil {
				log.Println(err)
			}
		}

		format := formatPool.Get()
		defer formatPool.Put(format)

		*format = (*format)[:0]
		for range msg.BaseS {
			*format = append(*format, '%', 'v', ' ')
		}
		if msg.format == "" {
			*format = append(*format, '%', 'v')
		} else {
			*format = append(*format, []byte(msg.format)...)
		}
		*format = append(*format, []byte(psys.EOL)...)

		val := valPool.Get()
		defer valPool.Put(val)
		*val = append((*val)[:0], msg.BaseS...)
		*val = append(*val, msg.msgs...)

		if prefix := msg.PrefixS[msg.prefix]; prefix != msg.logger.Prefix() {
			msg.logger.SetPrefix(prefix)
		}
		msg.logger.Printf(string(*format), *val...)
		return false
	})
	//启动阻塞
	o.MQ.PushLock_tag(`block`, &MsgItem{})
	return
}

func Copy(i *LogI) (o *LogI) {
	o = &LogI{
		Config: (*i).Config,
		MQ:     (*i).MQ,
	}
	return
}

// Level 设置之后日志等级
func (I *LogI) Level(log map[Level]string) (O *LogI) {
	O = Copy(I)
	for k, v := range log {
		if _, ok := O.PrefixS[k]; !ok {
			delete(O.PrefixS, k)
		}
		O.PrefixS[k] = v
	}
	return
}

func (I *LogI) LShow(show bool) (O *LogI) {
	O = Copy(I)
	O.NoStdout = !show
	O.reloadLogger()
	return
}

// Open 日志输出至DB
func (I *LogI) LDB(db *sql.DB, dBHolder psql.ReplaceF, insert string) (O *LogI) {
	O = Copy(I)
	if db != nil && insert != `` && dBHolder != nil {
		O.DBInsert = insert
		O.DBHolder = dBHolder
		O.dbPool = psql.NewTxPool[any](db)
		O.dbInsert = &psql.SqlFunc[any]{Sql: insert}
	} else {
		O.dbPool = nil
	}
	return
}

func (I *LogI) LFile(fileP string) (O *LogI) {
	O = Copy(I)
	if O.File != `` && fileP != `` {
		O.File = fileP
		f.New(O.File, 0, true).Create()
	} else {
		O.File = ``
	}
	O.reloadLogger()
	return
}

func (I *LogI) Close() {
	I.MQ.ClearAll()
	if I.DBConn != nil {
		(*I.DBConn).Close()
	}
}

// 日志等级
// Base 追加到后续输出
func (I *LogI) Base(i ...any) (O *LogI) {
	O = Copy(I)
	O.BaseS = i
	return
}
func (I *LogI) BaseAdd(i ...any) (O *LogI) {
	O = Copy(I)
	O.BaseS = append(O.BaseS, i...)
	return
}

var msgItemPool = pool.NewPoolBlock[MsgItem]()

func (I *LogI) LF(prefix Level, format string, i ...any) (O *LogI) {
	O = I
	if _, ok := O.PrefixS[prefix]; !ok {
		return
	}

	item := msgItemPool.Get()
	defer msgItemPool.Put(item)

	item.prefix = prefix
	item.format = format
	item.msgs = i
	item.Config = O.Config

	O.MQ.Push_tag(`L`, item)
	return
}
func (I *LogI) L(prefix Level, i ...any) (O *LogI) {
	return I.LF(prefix, "", i...)
}

func (Il *LogI) T(i ...any) (O *LogI) {
	return Il.L(T, i...)
}
func (Il *LogI) I(i ...any) (O *LogI) {
	return Il.L(I, i...)
}
func (Il *LogI) W(i ...any) (O *LogI) {
	return Il.L(W, i...)
}
func (Il *LogI) E(i ...any) (O *LogI) {
	return Il.L(E, i...)
}

func (Il *LogI) TF(format string, i ...any) (O *LogI) {
	return Il.LF(T, format, i...)
}
func (Il *LogI) IF(format string, i ...any) (O *LogI) {
	return Il.LF(I, format, i...)
}
func (Il *LogI) WF(format string, i ...any) (O *LogI) {
	return Il.LF(W, format, i...)
}
func (Il *LogI) EF(format string, i ...any) (O *LogI) {
	return Il.LF(E, format, i...)
}
