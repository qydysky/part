package part

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	pctx "github.com/qydysky/part/ctx"
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
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:         Execf,
		Ctx:        ctx,
		Sql:        "create table log (msg text)",
		SkipSqlErr: true,
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:         Execf,
		Ctx:        ctx,
		Sql:        "create table log2 (msg text)",
		SkipSqlErr: true,
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "delete from log",
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "delete from log2",
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:   Execf,
		Ctx:  ctx,
		Sql:  "insert into log values (?)",
		Args: []any{dateTime},
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:  Queryf,
		Ctx: ctx,
		Sql: "select msg from log",
	}).AfterQF(func(dataP *[]string, rows *sql.Rows, err *error) {
		names := make([]string, 0)
		for rows.Next() {
			var name string
			if *err = rows.Scan(&name); *err != nil {
				return
			}
			names = append(names, name)
		}
		rows.Close()

		if len(names) != 1 || dateTime != names[0] {
			*err = errors.New("no")
			return
		}

		*dataP = names
	})
	tx = tx.Do(&SqlFunc[[]string]{
		BeforeF: func(dataP *[]string, sqlf *SqlFunc[[]string], txE *error) {
			sqlf.Sql = "insert into log2 values (?)"
			sqlf.Args = append(sqlf.Args, (*dataP)[0])
		},
	})
	tx = tx.Do(&SqlFunc[[]string]{
		Ty:  Queryf,
		Ctx: ctx,
		Sql: "select msg from log2",
	}).AfterQF(func(dataP *[]string, rows *sql.Rows, err *error) {
		names := make([]string, 0)
		for rows.Next() {
			var name string
			if *err = rows.Scan(&name); *err != nil {
				return
			}
			names = append(names, name)
		}
		rows.Close()

		if len(names) != 1 || dateTime != names[0] {
			*err = errors.New("no2")
			return
		}

		*dataP = names
	})

	if _, e := tx.Fin(); e != nil {
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
	if _, e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (msg text)",
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
			x.Do(&SqlFunc[any]{
				Ty:   Execf,
				Sql:  "insert into log123 values (?)",
				Args: []any{"1"},
			})
			if _, e := x.Fin(); e != nil {
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
	// _ = file.Open("test.sqlite3").Delete()
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// defer func() {
	// 	_ = file.Open("test.sqlite3").Delete()
	// }()

	{
		tx := BeginTx[any](db, context.Background())
		tx.Do(&SqlFunc[any]{Sql: "create table log123 (msg INT,msg2 text)"})
		if _, e := tx.Fin(); e != nil {
			t.Fatal(e)
		}
	}

	type logg struct {
		Msg  int64
		Msg2 string
	}

	insertLog123 := SqlFunc[any]{Sql: "insert into log123 values ({Msg},{Msg2})"}
	{
		tx := BeginTx[any](db, context.Background())
		tx.DoPlaceHolder(&insertLog123, &logg{Msg: 1, Msg2: "a"}, PlaceHolderA)
		tx.DoPlaceHolder(&insertLog123, &logg{Msg: 2, Msg2: "b"}, PlaceHolderA)
		if _, e := tx.Fin(); e != nil {
			t.Fatal(e)
		}
		tx1 := BeginTx[any](db, context.Background()).SimplePlaceHolderA("insert into log123 values ({Msg},{Msg2})", &logg{Msg: 3, Msg2: "b"})
		if _, err := tx1.Fin(); err != nil {
			t.Fatal(err)
		}
	}
	{
		selectLog123 := SqlFunc[[]logg]{Sql: "select msg as Msg, msg2 as Msg2 from log123 where msg = {Msg}"}
		tx := BeginTx[[]logg](db, context.Background())
		tx.DoPlaceHolder(&selectLog123, &logg{Msg: 2, Msg2: "b"}, PlaceHolderA)
		tx.AfterQF(func(ctxVP *[]logg, rows *sql.Rows, txE *error) {
			*ctxVP, *txE = DealRows[logg](rows)
		})
		if v, e := tx.Fin(); e != nil {
			t.Fatal(e)
		} else {
			if v[0].Msg2 != "b" || v[0].Msg != 2 {
				t.Fatal(v[0])
			}
		}
	}
	{
		tx1 := BeginTx[[]logg](db, context.Background()).
			SimplePlaceHolderA("select msg as Msg, msg2 as Msg2 from log123 where msg2 = {Msg2}", &logg{Msg2: "b"}).
			AfterQF(func(ctxVP *[]logg, rows *sql.Rows, e *error) {
				*ctxVP, *e = DealRows[logg](rows)
			})
		if v, err := tx1.Fin(); err != nil {
			t.Fatal(err)
		} else {
			if v[0].Msg2 != "b" || v[0].Msg != 2 {
				t.Fatal()
			}
			if v[1].Msg2 != "b" || v[1].Msg != 3 {
				t.Fatal()
			}
		}
	}
}

func TestMain4(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "test.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer func() {
		_ = file.New("test.sqlite3", 0, true).Delete()
	}()

	conn, _ := db.Conn(context.Background())
	if _, e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (msg text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	tx1 := BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "insert into log123 values ('1')",
	})

	if _, e := tx1.Fin(); e != nil {
		t.Log(e)
	}

	if !IsFin(tx1) {
		t.Fatal()
	}
}

func Local_TestPostgresql(t *testing.T) {
	// connect
	db, err := sql.Open("pgx", "postgres://postgres:qydysky@192.168.31.103:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// c := pctx.CarryCancel(context.WithTimeout(context.Background(), time.Second))
	// if e := db.PingContext(c); e != nil {
	// 	t.Fatal(e)
	// }

	type test1 struct {
		Created string `sql:"sss"`
	}

	if _, e := BeginTx[any](db, pctx.GenTOCtx(time.Second), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Sql:        "create table test (created varchar(20))",
		SkipSqlErr: true,
	}).Fin(); e != nil {
		t.Fatal(e)
	}

	if _, e := BeginTx[any](db, context.Background(), &sql.TxOptions{}).DoPlaceHolder(&SqlFunc[any]{
		Sql: "insert into test (created) values ({Created})",
	}, &test1{"1"}, PlaceHolderB).Fin(); e != nil {
		t.Fatal(e)
	}

	if _, e := BeginTx[any](db, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Sql: "select created as sss from test",
		AfterQF: func(_ *any, rows *sql.Rows, txE *error) {
			if rowsP, e := DealRows[test1](rows); e != nil {
				*txE = e
			} else {
				if len(rowsP) != 1 {
					*txE = errors.New("no match")
					return
				}
				if rowsP[0].Created != "1" {
					*txE = errors.New("no match")
					return
				}
			}
		},
	}).Fin(); e != nil {
		t.Fatal(e)
	}
}

func Test1(t *testing.T) {
	if ToCamel("A_c") != "aC" {
		t.Fatal()
	}
	if ToCamel("A_C") != "aC" {
		t.Fatal()
	}
	if ToCamel("a_C") != "aC" {
		t.Fatal()
	}
	if ToCamel("a_") != "a" {
		t.Fatal()
	}
	if ToCamel("A_") != "a" {
		t.Fatal()
	}
	if ToCamel("_a") != "A" {
		t.Fatal()
	}
	if ToCamel("_A") != "A" {
		t.Fatal()
	}
	if ToCamel("_Aa") != "Aa" {
		t.Fatal()
	}
	if ToCamel("_aA") != "Aa" {
		t.Fatal()
	}
	if ToCamel("A好a") != "a好a" {
		t.Fatal()
	}
	if ToCamel("A好_a") != "a好A" {
		t.Fatal()
	}
}

func Test2(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	conn, _ := db.Conn(context.Background())
	if _, e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (msg_w text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	m := make(map[string]string)
	m["id"] = "123"

	x := BeginTx[any](db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values ({id})", &m)
	if _, e := x.Fin(); e != nil {
		t.Fatal(e)
	}

	{
		if _, err := BeginTx[any](db, context.Background()).
			SimplePlaceHolderA("select msg_w from log123", nil).
			AfterQF(func(ctxVP *any, rows *sql.Rows, e *error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["msgW"] != "123" {
						t.Fatal()
					}
				}
			}).Fin(); err != nil {
			t.Fatal(err)
		}
	}
}

func Test3(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	conn, _ := db.Conn(context.Background())
	if _, e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	x := BeginTx[any](db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if _, e := x.Fin(); e != nil {
		t.Fatal(e)
	}

	{
		if _, err := BeginTx[any](db, context.Background()).
			SimplePlaceHolderA("select a,b from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(ctxVP *any, rows *sql.Rows, e *error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
			}).Fin(); err != nil {
			t.Fatal(err)
		}
	}
}

func Test4(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	conn, _ := db.Conn(context.Background())
	if _, e := BeginTx[any](conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	x := BeginTx[any](db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if _, e := x.Fin(); e != nil {
		t.Fatal(e)
	}

	{
		if _, err := BeginTx[any](db, context.Background()).
			SimplePlaceHolderA("select a,d from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(ctxVP *any, rows *sql.Rows, e *error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
			}).Fin(); err != nil {
			if !errors.Is(err, ErrQuery) {
				t.Fatal()
			}
		}
	}
}

func Test5(t *testing.T) {
	err := error(&ErrTx[any]{})
	if _, ok := err.(interface{ Is(error) bool }); ok {
		t.Log("ok")
	} else {
		t.Fatal()
	}
}

func Test6(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	txPool := NewTxPool[any](db)
	if _, e := txPool.BeginTx(t.Context()).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Fin(); e != nil {
		t.Fatal(e)
	}

	x := txPool.BeginTx(t.Context())
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if _, e := x.Fin(); e != nil {
		t.Fatal(e)
	}

	{
		if _, err := txPool.BeginTx(t.Context()).
			SimplePlaceHolderA("select a,d from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(ctxVP *any, rows *sql.Rows, e *error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
			}).Fin(); err != nil {
			if !errors.Is(err, ErrQuery) {
				t.Fatal()
			}
		}
	}
}

func Test7(t *testing.T) {
	if rul := testing.Benchmark(Benchmark1); rul.AllocedBytesPerOp() > 1024 || rul.AllocsPerOp() > 22 {
		t.Fatal()
	}
}

// 18697 ns/op            1024 B/op         22 allocs/op
func Benchmark1(b *testing.B) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	txPool := NewTxPool[any](db)
	if _, e := txPool.BeginTx(context.Background()).Do(&SqlFunc[any]{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Fin(); e != nil {
		b.Fatal(e)
	}

	x := txPool.BeginTx(context.Background())
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if _, e := x.Fin(); e != nil {
		b.Fatal(e)
	}

	sqlF := &SqlFunc[any]{
		Ty:  Queryf,
		Sql: "select 1 from log123 where 1=0",
	}

	for b.Loop() {
		if _, err := txPool.BeginTx(context.Background()).Do(sqlF).Fin(); err != nil {
			b.Fatal(err)
		}
	}
}
