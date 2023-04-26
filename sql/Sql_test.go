package part

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	file "github.com/qydysky/part/file"
	_ "modernc.org/sqlite"
)

func TestMain(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	dateTime := time.Now().Format(time.DateTime)
	tx := BeginTx[[]string](db, ctx, &sql.TxOptions{})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:         Execf,
		Ctx:        ctx,
		Query:      "create table log (msg text)",
		SkipSqlErr: true,
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:         Execf,
		Ctx:        ctx,
		Query:      "create table log2 (msg text)",
		SkipSqlErr: true,
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:    Execf,
		Ctx:   ctx,
		Query: "delete from log",
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:    Execf,
		Ctx:   ctx,
		Query: "delete from log2",
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:    Execf,
		Ctx:   ctx,
		Query: "insert into log values (?)",
		Args:  []any{dateTime},
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:    Queryf,
		Ctx:   ctx,
		Query: "select msg from log",
		AfterQF: func(dataP *[]string, rows *sql.Rows, err error) (dataPR *[]string, stopErr error) {
			names := make([]string, 0)
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				names = append(names, name)
			}
			rows.Close()

			if len(names) != 1 || dateTime != names[0] {
				return nil, errors.New("no")
			}

			return &names, nil
		},
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:  Execf,
		Ctx: ctx,
		BeforeEF: func(dataP *[]string, sqlf *SqlFunc[[]string], txE error) (dataPR *[]string, stopErr error) {
			sqlf.Query = "insert into log2 values (?)"
			sqlf.Args = append(sqlf.Args, (*dataP)[0])
			return dataP, nil
		},
	})
	tx = tx.Do(SqlFunc[[]string]{
		Ty:    Queryf,
		Ctx:   ctx,
		Query: "select msg from log2",
		AfterQF: func(dataP *[]string, rows *sql.Rows, err error) (dataPR *[]string, stopErr error) {
			names := make([]string, 0)
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, err
				}
				names = append(names, name)
			}
			rows.Close()

			if len(names) != 1 || dateTime != names[0] {
				return nil, errors.New("no2")
			}

			return &names, nil
		},
	})

	if e := tx.Fin(); e != nil {
		t.Fatal(e)
	}
}

func TestMain2(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	conn, _ := db.Conn(context.Background())
	if e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "create table log123 (msg text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	var res = make(chan string, 101)
	var wg sync.WaitGroup
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			x := BeginTx[any](db, context.Background(), &sql.TxOptions{})
			x.Do(SqlFunc[any]{
				Ty:    Execf,
				Query: "insert into log123 values (?)",
				Args:  []any{"1"},
			})
			if e := x.Fin(); e != nil {
				res <- e.Error()
			}
			wg.Done()
		}()
	}

	wg.Wait()
	for len(res) > 0 {
		t.Fatal(<-res)
	}
}

func TestMain3(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "test.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer file.New("test.sqlite3", 0, true).Delete()

	conn, _ := db.Conn(context.Background())
	if e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "create table log123 (msg text,msg2 text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	tx1 := BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "insert into log123 values ('1','a')",
	})

	tx2 := BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "insert into log123 values ('2','b')",
	})

	if e := tx1.Fin(); e != nil {
		t.Log(e)
	}
	if e := tx2.Fin(); e != nil {
		t.Log(e)
	}

	tx1 = BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Queryf,
		Query: "select 1 as Msg, msg2 as Msg2 from log123",
		AfterQF: func(_ *any, rows *sql.Rows, txE error) (_ *any, stopErr error) {
			type logg struct {
				Msg  int64
				Msg2 string
			}

			if v, err := DealRows(rows, func() logg { return logg{} }); err != nil {
				return nil, err
			} else {
				if v[0].Msg2 != "a" {
					t.Fatal()
				}
				if v[1].Msg2 != "b" {
					t.Fatal()
				}
			}
			return
		},
	})
	if e := tx1.Fin(); e != nil {
		t.Fatal(e)
	}
}

func TestMain4(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "test.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer file.New("test.sqlite3", 0, true).Delete()

	conn, _ := db.Conn(context.Background())
	if e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "create table log123 (msg text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	tx1 := BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(SqlFunc[any]{
		Ty:    Execf,
		Query: "insert into log123 values ('1')",
	})

	if e := tx1.Fin(); e != nil {
		t.Log(e)
	}

	if !IsFin(tx1) {
		t.Fatal()
	}
}
