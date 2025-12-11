package part

import (
	// "fmt"

	"bytes"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	_ "net/http/pprof"

	_ "modernc.org/sqlite"

	pctx "github.com/qydysky/part/ctx"
	pf "github.com/qydysky/part/file"
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
		tx := psql.BeginTx(db, pctx.GenTOCtx(time.Second), &sql.TxOptions{})
		tx = tx.Do(&psql.SqlFunc{
			Sql: "create table log (p test,base text,msg text)",
		})
		if err := tx.Run(); err != nil {
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
		tx := psql.BeginTx(db, pctx.GenTOCtx(time.Second), &sql.TxOptions{})
		tx = tx.SimpleDo("select p,base,msg as s from log")
		tx.AfterQF(func(rows *sql.Rows) (e error) {
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
		if err := tx.Run(); err != nil {
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

func Test6(t *testing.T) {
	lf := pf.Open("1.log")
	lf.Delete()
	defer lf.Delete()

	logger := New(&Log{
		File: `1.log`,
	})
	logger.I("1", "2")
	logger.I("1", "3")
	if data, e := lf.ReadAll(10, 100); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(e)
	} else if bytes.Contains(data, []byte("%!(EXTRA")) {
		t.Fatal()
	}
}

func Test7(t *testing.T) {
	l0 := New(&Log{})
	l1 := l0.Level(map[Level]string{})
	if _, ok := l1.PrefixS[I]; ok {
		t.Fatal()
	}
	if _, ok := l0.PrefixS[I]; !ok {
		t.Fatal()
	}
}

func Test8(t *testing.T) {
	var buf strings.Builder
	l0 := New(&Log{NoStdout: true})
	l1 := l0.LShow(true).LOutput(&buf)
	if l1 != l0 {
		t.Fatal()
	}
}
