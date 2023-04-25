package part

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	Execf = iota
	Queryf
)

type CanTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

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

func BeginTx[T any](canTx CanTx, ctx context.Context, opts *sql.TxOptions) *SqlTx[T] {
	var sqlTX = SqlTx[T]{}

	if tx, err := canTx.BeginTx(ctx, opts); err != nil {
		sqlTX.err = fmt.Errorf("BeginTx; [] >> %s", err)
	} else {
		sqlTX.tx = tx
	}

	return &sqlTX
}

func (t *SqlTx[T]) Do(sqlf SqlFunc[T]) *SqlTx[T] {
	if t.err != nil {
		return t
	}

	if sqlf.Ctx == nil {
		sqlf.Ctx = context.Background()
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
				t.err = errors.Join(t.err, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
			}
		} else if sqlf.AfterEF != nil {
			if datap, err := sqlf.AfterEF(t.dataP, res, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
			} else {
				t.dataP = datap
			}
		}
	case Queryf:
		if sqlf.BeforeQF != nil {
			if datap, err := sqlf.BeforeQF(t.dataP, &sqlf, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
			} else {
				t.dataP = datap
			}
		}
		if res, err := t.tx.QueryContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
			if !sqlf.SkipSqlErr {
				t.err = errors.Join(t.err, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
			}
		} else if sqlf.AfterQF != nil {
			if datap, err := sqlf.AfterQF(t.dataP, res, t.err); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
			} else {
				t.dataP = datap
			}
		}
	}
	return t
}

func (t *SqlTx[T]) Fin() error {
	if t.err != nil {
		if t.tx != nil {
			if err := t.tx.Rollback(); err != nil {
				t.err = errors.Join(t.err, fmt.Errorf("Rollback; [] >> %s", err))
			}
		}
	} else {
		if err := t.tx.Commit(); err != nil {
			t.err = errors.Join(t.err, fmt.Errorf("Commit; [] >> %s", err))
		}
	}
	return t.err
}
