package part

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	null = iota
	Execf
	Queryf
)

type CanTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type BeforeF[T any] func(dataP *T, sqlf *SqlFunc[T], txE error) (dataPR *T, stopErr error)
type AfterEF[T any] func(dataP *T, result sql.Result, txE error) (dataPR *T, stopErr error)
type AfterQF[T any] func(dataP *T, rows *sql.Rows, txE error) (dataPR *T, stopErr error)

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
	beforeF    BeforeF[T]
	afterEF    AfterEF[T]
	afterQF    AfterQF[T]
}

func BeginTx[T any](canTx CanTx, ctx context.Context, opts ...*sql.TxOptions) *SqlTx[T] {
	var tx = SqlTx[T]{
		canTx: canTx,
		ctx:   ctx,
	}
	if len(opts) > 0 {
		tx.opts = opts[0]
	}
	return &tx
}

func (t *SqlTx[T]) Do(sqlf SqlFunc[T]) *SqlTx[T] {
	t.sqlFuncs = append(t.sqlFuncs, &sqlf)
	return t
}

func (t *SqlTx[T]) DoPlaceHolder(sqlf SqlFunc[T], ptr any) *SqlTx[T] {
	dataR := reflect.ValueOf(ptr).Elem()
	for i := 0; i < dataR.NumField(); i++ {
		field := dataR.Field(i)
		if field.IsValid() && field.CanSet() {
			replaceS := "{" + dataR.Type().Field(i).Name + "}"
			if strings.Contains(sqlf.Query, replaceS) {
				sqlf.Query = strings.ReplaceAll(sqlf.Query, replaceS, "?")
				sqlf.Args = append(sqlf.Args, field.Interface())
			}
		}
	}
	return t.Do(sqlf)
}

func (t *SqlTx[T]) BeforeF(f BeforeF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].beforeF = f
	}
	return t
}

func (t *SqlTx[T]) AfterEF(f AfterEF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].afterEF = f
	}
	return t
}

func (t *SqlTx[T]) AfterQF(f AfterQF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].afterQF = f
	}
	return t
}

func (t *SqlTx[T]) Fin() (dataP *T, e error) {
	if t.fin {
		return nil, fmt.Errorf("BeginTx; [] >> fin")
	}

	tx, err := t.canTx.BeginTx(t.ctx, t.opts)
	if err != nil {
		e = fmt.Errorf("BeginTx; [] >> %s", err)
	} else {
		for i := 0; i < len(t.sqlFuncs); i++ {
			sqlf := t.sqlFuncs[i]

			if sqlf.beforeF != nil {
				if datap, err := sqlf.beforeF(t.dataP, sqlf, e); err != nil {
					e = errors.Join(e, fmt.Errorf("%s; >> %s", sqlf.Query, err))
				} else {
					t.dataP = datap
				}
			}

			if strings.TrimSpace(sqlf.Query) == "" {
				continue
			}

			if sqlf.Ctx == nil {
				sqlf.Ctx = t.ctx
			}

			if sqlf.Ty == null {
				sqlf.Ty = Execf
				if uquery := strings.ToUpper(strings.TrimSpace(sqlf.Query)); strings.HasPrefix(uquery, "SELECT") {
					sqlf.Ty = Queryf
				}
			}

			switch sqlf.Ty {
			case Execf:
				if res, err := tx.ExecContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					}
				} else if sqlf.afterEF != nil {
					if datap, err := sqlf.afterEF(t.dataP, res, e); err != nil {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					} else {
						t.dataP = datap
					}
				}
			case Queryf:
				if res, err := tx.QueryContext(sqlf.Ctx, sqlf.Query, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						e = errors.Join(e, fmt.Errorf("%s; %s >> %s", sqlf.Query, sqlf.Args, err))
					}
				} else if sqlf.afterQF != nil {
					if datap, err := sqlf.afterQF(t.dataP, res, e); err != nil {
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
	return t.dataP, e
}

func IsFin[T any](t *SqlTx[T]) bool {
	return t == nil || t.fin
}

func DealRows[T any](rows *sql.Rows, createF func() T) (*[]T, error) {
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

		var stu = createF()
		for i := 0; i < len(rowNames); i++ {
			v := reflect.ValueOf(&stu).Elem().FieldByName(rowNames[i])
			if v.IsValid() {
				refT := reflect.TypeOf(&stu).Elem()
				if v.CanSet() {
					val := reflect.ValueOf(*rowP[i].(*any))
					if reflect.TypeOf(*rowP[i].(*any)).ConvertibleTo(v.Type()) {
						v.Set(val)
					} else {
						return nil, fmt.Errorf("DealRows:KindNotMatch:[sql] %v !> [%s.%s] %v", val.Kind(), refT.Name(), rowNames[i], v.Type())
					}
				} else {
					return nil, fmt.Errorf("DealRows:%s.%s CanSet:%v", refT.Name(), rowNames[i], v.CanSet())
				}
			}
		}
		res = append(res, stu)

	}

	return &res, nil
}

func SimpleQ[T any](canTx CanTx, query string, ptr *T) (*[]T, error) {
	tx := BeginTx[[]T](canTx, context.Background())
	tx.DoPlaceHolder(SqlFunc[[]T]{Query: query}, ptr)
	tx.AfterQF(func(_ *[]T, rows *sql.Rows, txE error) (dataPR *[]T, stopErr error) {
		if txE != nil {
			return nil, txE
		}
		return DealRows(rows, func() T { return *ptr })
	})
	return tx.Fin()
}