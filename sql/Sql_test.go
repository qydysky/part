package part

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	pctx "github.com/qydysky/part/ctx"
	_ "modernc.org/sqlite"
)

func TestMain8(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "./a")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("./a")
	defer db.Close()

	ctx := context.Background()
	a := BeginTx(db, ctx).Do(&SqlFunc{Sql: "select msg as Msg from danmu"})
	if a.sqlFuncs[0].Ty == null {
		t.Fatal()
	}
}

func TestMain6(t *testing.T) {
	// connect
	var e error
	if !HasErrTx(e, nil) {
		t.Fatal()
	}
}

func TestMain10(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "./a")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer os.Remove("./a")
	defer db.Close()

	ctx := context.Background()

	dbpool := NewTxPool(db)

	if e := dbpool.BeginTx(ctx).SimpleDo("create table log (msg text)").Run(); e != nil {
		t.Fatal(e)
	}

	tx1 := dbpool.BeginTx(ctx).SimpleDo("insert into log (msg) values ('1')")
	tx2 := dbpool.BeginTx(ctx).SimpleDo("insert into log (msg) values ('1')")

	e1 := tx1.do()
	e2 := tx2.do()
	t.Log(tx1.commitOrRollback(e1))
	t.Log(tx2.commitOrRollback(e2))
}

func TestMain9(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "./a")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer os.Remove("./a")
	defer db.Close()

	ctx := context.Background()

	dbpool := NewTxPool(db)

	if e := dbpool.BeginTx(ctx).SimpleDo("create table log (msg text)").Run(); e != nil {
		t.Fatal(e)
	}

	n := 1000
	now := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if e := dbpool.BeginTx(ctx).SimpleDo("insert into log (msg) values ('1')").Run(); e != nil {
				t.Fatal(e)
			}
		}()
	}
	wg.Wait()
	t.Log(time.Since(now).Milliseconds() / int64(n))
}

func TestMain7(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", "./a")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer os.Remove("./a")
	defer db.Close()

	ctx := context.Background()

	if e := BeginTx(db, ctx).SimpleDo("create table log (msg text)").Run(); e != nil {
		t.Fatal(e)
	}

	n := 1000
	now := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if e := BeginTx(db, ctx).SimpleDo("insert into log (msg) values ('1')").Run(); e != nil {
				t.Fatal(e)
			}
		}()
	}
	wg.Wait()
	t.Log(time.Since(now).Milliseconds() / int64(n))
}

func TestMain5(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	tx := BeginTx(db, ctx)
	tx = tx.Do(&SqlFunc{
		Sql: "create table log (msg text)",
	})
	tx = tx.Do(&SqlFunc{
		Sql: "create table log (msg text)",
	})
	if e := tx.Run(); !strings.Contains(e.Error(), "table log already exists") {
		t.Fatal(e)
	}
	if e := BeginTx(db, ctx).SimpleDo("create table log (msg text)").Run(); e != nil {
		t.Fatal(e)
	}
}

func TestMain(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	dateTime := time.Now().Format(time.DateTime)
	tx := BeginTx(db, ctx, &sql.TxOptions{})
	tx = tx.Do(&SqlFunc{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "create table log (msg text)",
	})
	tx = tx.Do(&SqlFunc{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "create table log2 (msg text)",
	})
	tx = tx.Do(&SqlFunc{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "delete from log",
	})
	tx = tx.Do(&SqlFunc{
		Ty:  Execf,
		Ctx: ctx,
		Sql: "delete from log2",
	})
	tx = tx.Do(&SqlFunc{
		Ty:   Execf,
		Ctx:  ctx,
		Sql:  "insert into log values (?)",
		Args: []any{dateTime},
	})
	var dataP []string
	tx = tx.Do(&SqlFunc{
		Ty:  Queryf,
		Ctx: ctx,
		Sql: "select msg from log",
	}).AfterQF(func(rows *sql.Rows) (err error) {
		names := make([]string, 0)
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				return
			}
			names = append(names, name)
		}
		rows.Close()

		if len(names) != 1 || dateTime != names[0] {
			err = errors.New("no")
			return
		}

		dataP = names
		return nil
	})
	tx = tx.Do(&SqlFunc{
		BeforeF: func(sqlf *SqlFunc) error {
			sqlf.Sql = "insert into log2 values (?)"
			sqlf.Args = append(sqlf.Args, dataP[0])
			return nil
		},
	})
	tx = tx.Do(&SqlFunc{
		Ty:  Queryf,
		Ctx: ctx,
		Sql: "select msg from log2",
	}).AfterQF(func(rows *sql.Rows) (err error) {
		names := make([]string, 0)
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				return
			}
			names = append(names, name)
		}
		rows.Close()

		if len(names) != 1 || dateTime != names[0] {
			return errors.New("no2")
		}

		dataP = names
		return nil
	})

	if e := tx.Run(); e != nil {
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
	if e := BeginTx(conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (msg text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	var res = make(chan string, 101)
	var wg sync.WaitGroup
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			x := BeginTx(db, context.Background(), &sql.TxOptions{})
			x.Do(&SqlFunc{
				Ty:   Execf,
				Sql:  "insert into log123 values (?)",
				Args: []any{"1"},
			})
			if e := x.Run(); e != nil {
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
		tx := BeginTx(db, context.Background())
		tx.Do(&SqlFunc{Sql: "create table log123 (msg INT,msg2 text)"})
		if e := tx.Run(); e != nil {
			t.Fatal(e)
		}
	}

	type logg struct {
		Msg  int64
		Msg2 string
	}

	insertLog123 := SqlFunc{Sql: "insert into log123 values ({Msg},{Msg2})"}
	{
		tx := BeginTx(db, context.Background())
		tx.DoPlaceHolder(&insertLog123, &logg{Msg: 1, Msg2: "a"}, PlaceHolderA)
		tx.DoPlaceHolder(&insertLog123, &logg{Msg: 2, Msg2: "b"}, PlaceHolderA)
		if e := tx.Run(); e != nil {
			t.Fatal(e)
		}
		tx1 := BeginTx(db, context.Background()).SimplePlaceHolderA("insert into log123 values ({Msg},{Msg2})", &logg{Msg: 3, Msg2: "b"})
		if err := tx1.Run(); err != nil {
			t.Fatal(err)
		}
	}
	{
		selectLog123 := SqlFunc{Sql: "select msg as Msg, msg2 as Msg2 from log123 where msg = {Msg}"}
		tx := BeginTx(db, context.Background())
		tx.DoPlaceHolder(&selectLog123, &logg{Msg: 2, Msg2: "b"}, PlaceHolderA)
		var ctxVP []logg
		tx.AfterQF(func(rows *sql.Rows) (e error) {
			ctxVP, e = DealRows[logg](rows)
			return
		})
		if e := tx.Run(); e != nil {
			t.Fatal(e)
		} else {
			if ctxVP[0].Msg2 != "b" || ctxVP[0].Msg != 2 {
				t.Fatal(ctxVP[0])
			}
		}
	}
	{
		var ctxVP []logg
		tx1 := BeginTx(db, context.Background()).
			SimplePlaceHolderA("select msg as Msg, msg2 as Msg2 from log123 where msg2 = {Msg2}", &logg{Msg2: "b"}).
			AfterQF(func(rows *sql.Rows) (e error) {
				ctxVP, e = DealRows[logg](rows)
				return
			})
		if err := tx1.Run(); err != nil {
			t.Fatal(err)
		} else {
			if ctxVP[0].Msg2 != "b" || ctxVP[0].Msg != 2 {
				t.Fatal()
			}
			if ctxVP[1].Msg2 != "b" || ctxVP[1].Msg != 3 {
				t.Fatal()
			}
		}
	}
}

func TestMain4(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	if e := BeginTx(db, context.Background()).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (msg text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}

	tx1 := BeginTx(db, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "insert into log123 values ('1')",
	})

	if e := tx1.Run(); e != nil {
		t.Log(e)
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

	if e := BeginTx(db, pctx.GenTOCtx(time.Second), &sql.TxOptions{}).Do(&SqlFunc{
		Sql: "create table test (created varchar(20))",
	}).Run(); e != nil {
		t.Fatal(e)
	}

	if e := BeginTx(db, context.Background(), &sql.TxOptions{}).DoPlaceHolder(&SqlFunc{
		Sql: "insert into test (created) values ({Created})",
	}, &test1{"1"}, PlaceHolderB).Run(); e != nil {
		t.Fatal(e)
	}

	if e := BeginTx(db, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Sql: "select created as sss from test",
		AfterQF: func(rows *sql.Rows) (txE error) {
			if rowsP, e := DealRows[test1](rows); e != nil {
				txE = e
			} else {
				if len(rowsP) != 1 {
					txE = errors.New("no match")
					return
				}
				if rowsP[0].Created != "1" {
					txE = errors.New("no match")
					return
				}
			}
			return
		},
	}).Run(); e != nil {
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
	if e := BeginTx(conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (msg_w text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	m := make(map[string]string)
	m["id"] = "123"

	x := BeginTx(db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values ({id})", &m)
	if e := x.Run(); e != nil {
		t.Fatal(e)
	}

	{
		if err := BeginTx(db, context.Background()).
			SimplePlaceHolderA("select msg_w from log123", nil).
			AfterQF(func(rows *sql.Rows) (e error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["msgW"] != "123" {
						t.Fatal()
					}
				}
				return
			}).Run(); err != nil {
			t.Fatal(err)
		}
	}
}

func Test9(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, _ := db.Conn(context.Background())
	defer conn.Close()

	tx := BeginTx(conn, context.Background())
	tx.SimpleDo("create table log123 (msg_w text)")
	if e := tx.Run(); e != nil {
		t.Fatal(e)
	}

	tx = BeginTx(conn, context.Background())
	tx.SimpleDo("insert into log123 values ('1')")
	tx.Do(&SqlFunc{
		Sql: "select count(*) c from log123",
		AfterQF: func(rows *sql.Rows) error {
			for v := range DealRowsMapIter(rows) {
				if v.Raw["c"].(int64) != 1 {
					return errors.New("1")
				}
				break
			}
			return nil
		},
	})
	tx.StopWithErr(errors.New("a"))
	if e := tx.Run(); e != nil {
		t.Log(e)
	}

	tx = BeginTx(conn, context.Background())
	tx.Do(&SqlFunc{
		Sql: "select count(*) c from log123",
		AfterQF: func(rows *sql.Rows) error {
			for v := range DealRowsMapIter(rows) {
				if v.Raw["c"].(int64) != 0 {
					return errors.New("1")
				}
				break
			}
			return nil
		},
	})
	if e := tx.Run(); e != nil {
		t.Fatal(e)
	}
}

func Test10(t *testing.T) {
	defer os.Remove("/tmp/b")
	db, err := sql.Open("sqlite", "/tmp/b")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	BeginTx(db, context.Background()).SimpleDo("create table log123 (msg_w text)").Run()

	txs := NewSqlTxs()

	BeginTx(db, context.Background()).AddToTxs(txs).SimpleDo("insert into log123 values ('2')")
	BeginTx(db, context.Background(), &sql.TxOptions{Isolation: sql.LevelReadUncommitted, ReadOnly: true}).AddToTxs(txs).SimpleDo("select count(1) c from log123").AfterQF(func(rows *sql.Rows) error {
		t.Log(DealRow[struct {
			C int64
		}](rows).Raw.C)
		return nil
	})

	t.Log(txs.Run())
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
	if e := BeginTx(conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	x := BeginTx(db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if e := x.Run(); e != nil {
		t.Fatal(e)
	}

	{
		if err := BeginTx(db, context.Background()).
			SimplePlaceHolderA("select a,b from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(rows *sql.Rows) (e error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
				return
			}).Run(); err != nil {
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
	if e := BeginTx(conn, context.Background(), &sql.TxOptions{}).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}
	conn.Close()

	x := BeginTx(db, context.Background(), &sql.TxOptions{})
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if e := x.Run(); e != nil {
		t.Fatal(e)
	}

	{
		if err := BeginTx(db, context.Background()).
			SimplePlaceHolderA("select a,d from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(rows *sql.Rows) (e error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
				return
			}).Run(); err != nil {
			if !errors.Is(err, ErrQuery) {
				t.Fatal()
			}
		}
	}
}

func Test5(t *testing.T) {
	err := error(&ErrTx{})
	if _, ok := err.(interface{ Is(error) bool }); ok {
		t.Log("ok")
	} else {
		t.Fatal()
	}
}

func Test8(t *testing.T) {
	txe := NewErrTx(nil, ErrBeforeF, errors.New("1"))
	txe = NewErrTx(txe, ErrAfterQuery, errors.New("2"))
	if !errors.Is(txe, ErrBeforeF) {
		t.Fatal()
	}
	if !errors.Is(txe, ErrAfterQuery) {
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

	txPool := NewTxPool(db)
	if e := txPool.BeginTx(t.Context()).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Run(); e != nil {
		t.Fatal(e)
	}

	x := txPool.BeginTx(t.Context())
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if e := x.Run(); e != nil {
		t.Fatal(e)
	}

	{
		if err := txPool.BeginTx(t.Context()).
			SimplePlaceHolderA("select a,d from log123 where c = {c}", &map[string]any{"c": "3"}).
			AfterQF(func(rows *sql.Rows) (e error) {
				for v := range DealRowsMapIter(rows, ToCamel) {
					if v.Err != nil {
						t.Fatal(v.Err)
					} else if v.Raw["a"] != int64(1) {
						t.Fatal(v.Raw["a"])
					} else if t1, e := time.Parse("2006-01-02", "2025-01-01"); e != nil || !t1.Equal(v.Raw["b"].(time.Time)) {
						t.Fatal(v.Raw["b"])
					}
				}
				return
			}).Run(); err != nil {
			if !errors.Is(err, ErrQuery) {
				t.Fatal()
			}
		}
	}
}
