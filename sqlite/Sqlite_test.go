package part

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMain(t *testing.T) {
	// connect
	db, err := sql.Open("sqlite3", ":memory:")
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
