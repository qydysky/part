package part

import (
	// "fmt"

	"database/sql"
	"errors"
	"log/slog"
	"testing"
	"time"

	_ "net/http/pprof"

	_ "modernc.org/sqlite"

	pctx "github.com/qydysky/part/ctx"
	psql "github.com/qydysky/part/sql"
)

func Test_1(t *testing.T) {
	n := New(&Log{
		File: `1.log`,
	})

	n.L(T, `s`).L(I, `s`)
	n.LFile(`2.log`).L(W, `s`).L(E, `s`)

	{
		n1 := n.Base(`>1`)
		n1.L(T, `s`).L(I, `s`)
		{
			n2 := n1.BaseAdd(`>2`)
			n2.L(T, `s`).L(I, `s`)
		}
	}

	n.Level(map[Level]string{W: "W:"}).L(T, `s`).L(I, `s`).L(W, `s`).L(E, `s`)
}

var n *Log

func Test_2(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	{
		tx := psql.BeginTx[any](db, pctx.GenTOCtx(time.Second), &sql.TxOptions{})
		tx = tx.Do(&psql.SqlFunc[any]{
			Sql:        "create table log (p test,base text,msg text)",
			SkipSqlErr: true,
		})
		if _, err := tx.Fin(); err != nil {
			t.Fatal(err)
		}
	}

	n = New(&Log{
		File: `1.log`,
	})

	ndb := n.BaseAdd(`>1`)
	ndb = ndb.LDB(db, psql.PlaceHolderA, `insert into log (p,base,msg) values ({Prefix},{Base},{Msgs})`)
	ndb.L(T, `s`)
	n.L(T, `p`)

	{
		type logg struct {
			P    string `sql:"p"`
			Base string
			Msg  string `sql:"s"`
		}
		tx := psql.BeginTx[any](db, pctx.GenTOCtx(time.Second), &sql.TxOptions{})
		tx = tx.SimpleDo("select p,base,msg as s from log")
		tx.AfterQF(func(_ *any, rows *sql.Rows) (e error) {
			if ls, err := psql.DealRows[logg](rows); err == nil {
				if len(ls) != 1 {
					return errors.New("num wrong")
				}
				if ls[0].Msg != "s" {
					return errors.New("msg wrong")
				}
			} else {
				return err
			}
			return nil
		})
		if _, err := tx.Fin(); err != nil {
			t.Fatal(err)
		}
	}
}

func Test_3(t *testing.T) {
	logger := slog.Default()
	logger = logger.WithGroup("122")
	logger.Info("sss", slog.String("1", "3"))
}

func Test4(t *testing.T) {
	rul := testing.Benchmark(Benchmark)
	if rul.AllocedBytesPerOp() > 98 || rul.AllocsPerOp() > 2 {
		t.Fatal()
	}
}

func Benchmark(b *testing.B) {
	logger := New(&Log{
		NoStdout: true,
	})
	for b.Loop() {
		logger.I("1")
	}
}

func Test5(t *testing.T) {
	logger := New(&Log{})
	logger.I("1")
}
