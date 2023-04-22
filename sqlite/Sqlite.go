package part

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Execf = iota
	Queryf
)

type SqlTx[T any] struct {
	tx    *sql.Tx
	dataP *T
	err   error
}

type SqlFunc[T any] struct {
	Ty         int
	Ctx        context.Context
	Query      string
	Args       []any
	SkipSqlErr bool
	BeforeEF   func(dataP *T, sqlf *SqlFunc[T], txE error) (dataPR *T, stopErr error)
	BeforeQF   func(dataP *T, sqlf *SqlFunc[T], txE error) (dataPR *T, stopErr error)
	AfterEF    func(dataP *T, result sql.Result, txE error) (dataPR *T, stopErr error)
	AfterQF    func(dataP *T, rows *sql.Rows, txE error) (dataPR *T, stopErr error)
}

func BeginTx[T any](db *sql.DB, ctx context.Context, opts *sql.TxOptions) *SqlTx[T] {
	var sqlTX = SqlTx[T]{}

	if tx, e := db.BeginTx(ctx, opts); e != nil {
		sqlTX.err = e
	} else {
		sqlTX.tx = tx
	}
	return &sqlTX
}

func (t *SqlTx[T]) Do(sqlf SqlFunc[T]) *SqlTx[T] {
	if t.err != nil {
		return t
	}

	switch sqlf.Ty {
	case Execf:
		if sqlf.BeforeEF != nil {
			if datap, err := sqlf.BeforeEF(t.dataP, &sqlf, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			} else {
				t.dataP = datap
			}
		}
		if res, err := t.tx.ExecContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
			if !sqlf.SkipSqlErr {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			}
		} else if sqlf.AfterEF != nil {
			if datap, err := sqlf.AfterEF(t.dataP, res, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			} else {
				t.dataP = datap
			}
		}
	case Queryf:
		if sqlf.BeforeQF != nil {
			if datap, err := sqlf.BeforeQF(t.dataP, &sqlf, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			} else {
				t.dataP = datap
			}
		}
		if res, err := t.tx.QueryContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
			if !sqlf.SkipSqlErr {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			}
		} else if sqlf.AfterQF != nil {
			if datap, err := sqlf.AfterQF(t.dataP, res, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s >> %s", sqlf.Query, err))
			} else {
				t.dataP = datap
			}
		}
	}
	return t
}

func (t *SqlTx[T]) Fin() error {
	if t.err != nil {
		return errors.Join(t.err, t.tx.Rollback())
	} else {
		return errors.Join(t.err, t.tx.Commit())
	}
}
