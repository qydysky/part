package part

import (
	// "fmt"

	"context"
	"database/sql"
	"errors"
	"testing"

	_ "net/http/pprof"

	_ "modernc.org/sqlite"

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
		tx := psql.BeginTx[any](db, context.Background(), &sql.TxOptions{})
		tx = tx.Do(psql.SqlFunc[any]{
			Query:      "create table log (p test,base text,msg text)",
			SkipSqlErr: true,
		})
		if _, err := tx.Fin(); err != nil {
			t.Fatal(err)
		}
	}

	n = New(Config{
		File:          `1.log`,
		Stdout:        true,
		Prefix_string: map[string]struct{}{`T:`: On, `I:`: On, `W:`: On, `E:`: On},
	})

	ndb := n.Base_add(`>1`)
	ndb = ndb.LDB(db, `insert into log (p,base,msg) values (?,?,?)`)
	ndb.L(`T:`, `s`)
	n.L(`T:`, `p`)

	{
		type logg struct {
			P    string `sql:"p"`
			Base string
			Msg  string `sql:"s"`
		}
		tx := psql.BeginTx[any](db, context.Background(), &sql.TxOptions{})
		tx = tx.SimpleDo("select p,base,msg as s from log")
		tx.AfterQF(func(_ *any, rows *sql.Rows, e *error) {
			if ls, err := psql.DealRows[logg](rows, func() logg { return logg{} }); err == nil {
				if len(ls) != 1 {
					*e = errors.New("num wrong")
				}
				if ls[0].Msg != "s" {
					*e = errors.New("msg wrong")
				}
			} else {
				*e = err
			}
		})
		if _, err := tx.Fin(); err != nil {
			t.Fatal(err)
		}
	}
}
