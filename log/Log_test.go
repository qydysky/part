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
	n := New(Config{
		File:          `1.log`,
		Stdout:        true,
		Prefix_string: map[string]struct{}{`T:`: On, `I:`: On, `W:`: On, `E:`: On},
	})

	n.L(`T:`, `s`).L(`I:`, `s`)
	n.Log_to_file(`2.log`).L(`W:`, `s`).L(`E:`, `s`)

	{
		n1 := n.Base(`>1`)
		n1.L(`T:`, `s`).L(`I:`, `s`)
		{
			n2 := n1.Base_add(`>2`)
			n2.L(`T:`, `s`).L(`I:`, `s`)
		}
	}

	n.Level(map[string]struct{}{`W:`: On}).L(`T:`, `s`).L(`I:`, `s`).L(`W:`, `s`).L(`E:`, `s`)
}

var n *Log_interface

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
		if err := tx.Run(); psql.HasErrTx(err, psql.ErrBeginTx) {
			t.Fatal(err)
		}
	}

	n = New(Config{
		File:          `1.log`,
		Stdout:        true,
		Prefix_string: map[string]struct{}{`T:`: On, `I:`: On, `W:`: On, `E:`: On},
	})

	ndb := n.Base_add(`>1`)
	ndb = ndb.LDB("sqlite", db, `insert into log (p,base,msg) values ({Prefix},{Base},{Msgs})`)
	ndb.L(`T:`, `s`)
	n.L(`T:`, `p`)

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
			return
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
