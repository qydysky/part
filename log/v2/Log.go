package part

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	f "github.com/qydysky/part/file"
	pool "github.com/qydysky/part/pool"
	psql "github.com/qydysky/part/sql"
	psys "github.com/qydysky/part/sys"
)

type Log struct {
	Output io.Writer
	File   string

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

	startLog atomic.Bool
}

func (o *Log) reloadLogger() {
	// logger
	var showObj = []io.Writer{}
	if o.Output != nil {
		showObj = append(showObj, o.Output)
	}
	if !o.NoStdout {
		showObj = append(showObj, os.Stdout)
	}
	if o.File != `` {
		file := f.New(o.File, -1, true)
		// file, err := os.OpenFile(o.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		// if err == nil {
		showObj = append(showObj, file)
		// defer file.Close()
		// } else {
		// log.Println(err)
		// }
	}
	o.logger = log.New(io.MultiWriter(showObj...), "", log.Ldate|log.Ltime)
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
func New(c *Log) (o *Log) {
	o = c
	if c.File != `` {
		f.New(c.File, 0, true).Create()
	}
	if o.DBConn != nil && o.DBInsert != `` && o.DBHolder != nil {
		o.dbPool = psql.NewTxPool[any](o.DBConn)
		o.dbInsert = &psql.SqlFunc[any]{Sql: o.DBInsert}
	}
	if o.PrefixS == nil {
		o.PrefixS = map[Level]string{T: "T:", I: "I:", W: "W:", E: "E:"}
	}

	o.reloadLogger()
	return
}

func Copy(i *Log) (o *Log) {
	o = &Log{
		File:     i.File,
		DBInsert: i.DBInsert,
		DBHolder: i.DBHolder,
		DBConn:   i.DBConn,
		dbPool:   i.dbPool,
		dbInsert: i.dbInsert,
		NoStdout: i.NoStdout,
		PrefixS:  maps.Clone(i.PrefixS),
		BaseS:    slices.Clone(i.BaseS),
		logger:   i.logger,
	}
	return
}

// Level 设置之后日志等级
func (I *Log) Level(log map[Level]string) (O *Log) {
	if I.startLog.Load() {
		O = Copy(I)
	} else {
		O = I
	}
	O.PrefixS = log
	return
}

func (I *Log) LOutput(o io.Writer) (O *Log) {
	if I.startLog.Load() {
		O = Copy(I)
	} else {
		O = I
	}
	O.Output = o
	return
}

func (I *Log) LShow(show bool) (O *Log) {
	if I.startLog.Load() {
		O = Copy(I)
	} else {
		O = I
	}
	O.NoStdout = !show
	O.reloadLogger()
	return
}

// Open 日志输出至DB
func (I *Log) LDB(db *sql.DB, dBHolder psql.ReplaceF, insert string) (O *Log) {
	if I.startLog.Load() {
		O = Copy(I)
	} else {
		O = I
	}
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

func (I *Log) LFile(fileP string) (O *Log) {
	if I.startLog.Load() {
		O = Copy(I)
	} else {
		O = I
	}
	if O.File != `` && fileP != `` {
		O.File = fileP
		f.New(O.File, 0, true).Create()
	} else {
		O.File = ``
	}
	O.reloadLogger()
	return
}

func (I *Log) Close() {
	if I.DBConn != nil {
		(*I.DBConn).Close()
	}
}

// 日志等级
// Base 追加到后续输出
func (I *Log) Base(i ...any) (O *Log) {
	O = Copy(I)
	O.BaseS = i
	return
}
func (I *Log) BaseAdd(i ...any) (O *Log) {
	O = Copy(I)
	O.BaseS = append(O.BaseS, i...)
	return
}

var (
	formatPool = pool.NewPoolBlocks(func() *[]byte {
		return &[]byte{'%', 'v', ' ', '%', 'v', ' '}
	})
	valPool = pool.NewPoolBlocks[any]()
)

func (I *Log) LF(prefix Level, formatS string, i ...any) (O *Log) {
	O = I
	if _, ok := O.PrefixS[prefix]; !ok {
		return
	}
	_ = O.startLog.CompareAndSwap(false, true)

	{
		if O.dbPool != nil {
			var sqlTx = O.dbPool.BeginTx(context.Background())
			sqlTx.DoPlaceHolder(O.dbInsert, &LogDb{
				Date:   time.Now().Format(time.DateTime),
				Unix:   time.Now().Unix(),
				Prefix: strings.TrimSpace(O.PrefixS[prefix]),
				Base:   strings.TrimSpace(fmt.Sprintln(O.BaseS...)),
				Msgs:   strings.TrimSpace(fmt.Sprintln(i...)),
			}, O.DBHolder)
			if _, err := sqlTx.Fin(); err != nil {
				log.Println(err)
			}
		}

		format := formatPool.Get()
		defer formatPool.Put(format)

		*format = (*format)[:0]
		for range O.BaseS {
			*format = append(*format, '%', 'v', ' ')
		}
		if formatS == "" {
			for j := 0; j < len(i); j++ {
				*format = append(*format, '%', 'v')
				if j < len(i)-1 {
					*format = append(*format, ' ')
				}
			}
		} else {
			*format = append(*format, []byte(formatS)...)
		}
		*format = append(*format, []byte(psys.EOL)...)

		val := valPool.Get()
		defer valPool.Put(val)

		*val = append((*val)[:0], O.BaseS...)
		*val = append(*val, i...)

		if prefix := O.PrefixS[prefix]; prefix != O.logger.Prefix() {
			O.logger.SetPrefix(prefix)
		}
		O.logger.Printf(string(*format), *val...)
	}
	return
}
func (I *Log) L(prefix Level, i ...any) (O *Log) {
	return I.LF(prefix, "", i...)
}

func (Il *Log) T(i ...any) (O *Log) {
	return Il.L(T, i...)
}
func (Il *Log) I(i ...any) (O *Log) {
	return Il.L(I, i...)
}
func (Il *Log) W(i ...any) (O *Log) {
	return Il.L(W, i...)
}
func (Il *Log) E(i ...any) (O *Log) {
	return Il.L(E, i...)
}

func (Il *Log) TF(format string, i ...any) (O *Log) {
	return Il.LF(T, format, i...)
}
func (Il *Log) IF(format string, i ...any) (O *Log) {
	return Il.LF(I, format, i...)
}
func (Il *Log) WF(format string, i ...any) (O *Log) {
	return Il.LF(W, format, i...)
}
func (Il *Log) EF(format string, i ...any) (O *Log) {
	return Il.LF(E, format, i...)
}
