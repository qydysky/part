package part

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

const (
	Execf = iota
	Queryf
)

type CanTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type SqlTx[T any] struct {
	canTx    CanTx
	ctx      context.Context
	opts     *sql.TxOptions
	sqlFuncs []*SqlFunc[T]
	dataP    *T
	fin      bool
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
	return &SqlTx[T]{
		canTx: canTx,
		ctx:   ctx,
		opts:  opts,
	}
}

func (t *SqlTx[T]) Do(sqlf SqlFunc[T]) *SqlTx[T] {
	t.sqlFuncs = append(t.sqlFuncs, &sqlf)
	return t
}

func (t *SqlTx[T]) Fin() (e error) {
	if t.fin {
		return fmt.Errorf("BeginTx; [] >> fin")
	}

	tx, err := t.canTx.BeginTx(t.ctx, t.opts)
	if err != nil {
		e = fmt.Errorf("BeginTx; [] >> %s", err)
	} else {
		for i := 0; i < len(t.sqlFuncs); i++ {
			sqlf := t.sqlFuncs[i]
			if sqlf.Ctx == nil {
				sqlf.Ctx = t.ctx
			}
			switch sqlf.Ty {
			case Execf:
				if sqlf.BeforeEF != nil {
					if datap, err := sqlf.BeforeEF(t.dataP, sqlf, e); err != nil {
						e = errors.Join(e, fmt.Errorf("%s >> %s", sqlf.Query, err))
					} else {
						t.dataP = datap
					}
				}
				if res, err := tx.ExecContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					}
				} else if sqlf.AfterEF != nil {
					if datap, err := sqlf.AfterEF(t.dataP, res, e); err != nil {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					} else {
						t.dataP = datap
					}
				}
			case Queryf:
				if sqlf.BeforeQF != nil {
					if datap, err := sqlf.BeforeQF(t.dataP, sqlf, e); err != nil {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					} else {
						t.dataP = datap
					}
				}
				if res, err := tx.QueryContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					}
				} else if sqlf.AfterQF != nil {
					if datap, err := sqlf.AfterQF(t.dataP, res, e); err != nil {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					} else {
						t.dataP = datap
					}
				}
			}
		}
	}
	if e != nil {
		if tx != nil {
			if err := tx.Rollback(); err != nil {
				e = errors.Join(e, fmt.Errorf("Rollback; [] >> %s", err))
			}
		}
	} else {
		if err := tx.Commit(); err != nil {
			e = errors.Join(e, fmt.Errorf("Commit; [] >> %s", err))
		}
	}
	t.fin = true
	return e
}

func IsFin[T any](t *SqlTx[T]) bool {
	return t == nil || t.fin
}

func DealRows[T any](rows *sql.Rows, createF func() T) ([]T, error) {
	rowNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var res []T

	for rows.Next() {
		rowP := make([]any, len(rowNames))
		for i := 0; i < len(rowNames); i++ {
			rowP[i] = new(any)
		}

		err = rows.Scan(rowP...)
		if err != nil {
			return nil, err
		}

		var rowT = createF()
		refV := reflect.ValueOf(&rowT)
		for i := 0; i < len(rowNames); i++ {
			v := refV.Elem().FieldByName(rowNames[i])
			if v.IsValid() {
				if v.CanSet() {
					val := reflect.ValueOf(*rowP[i].(*any))
					if val.Kind() == v.Kind() {
						v.Set(val)
					} else {
						return nil, fmt.Errorf("reflectFail:%s KindNotMatch:%v !> %v", rowNames[i], val.Kind(), v.Kind())
					}
				} else {
					return nil, fmt.Errorf("reflectFail:%s CanSet:%v", rowNames[i], v.CanSet())
				}
			}
		}
		res = append(res, rowT)

	}

	return res, nil
}
