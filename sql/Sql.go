package part

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
	"weak"

	pool "github.com/qydysky/part/pool"
	ps "github.com/qydysky/part/slice"
)

const (
	null Type = iota
	Execf
	Queryf
)

type Type int

type CanTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type BeforeF[T any] func(ctxVP *T, sqlf *SqlFunc[T], e *error)
type AfterEF[T any] func(ctxVP *T, result sql.Result, e *error)

// func(ctxVP *T, rows *sql.Rows, e *error)
type AfterQF[T any] func(ctxVP *T, rows *sql.Rows, e *error)

type SqlTx[T any] struct {
	canTx    CanTx
	ctx      context.Context
	opts     *sql.TxOptions
	sqlFuncs []*SqlFunc[T]
	fin      bool
	pool     weak.Pointer[TxPool[T]]
}

type SqlFunc[T any] struct {
	// 	Execf or Queryf, default: auto detection
	Ty Type
	// default: use transaction Ctx
	Ctx        context.Context
	Sql        string
	Args       []any
	SkipSqlErr bool
	BeforeF    BeforeF[T]
	AfterEF    AfterEF[T]
	AfterQF    AfterQF[T]
}

func (t *SqlFunc[T]) Clear() {
	t.Ty = null
	t.Ctx = context.Background()
	t.Sql = ""
	t.Args = t.Args[:0]
	t.SkipSqlErr = false
	t.BeforeF = nil
	t.AfterEF = nil
	t.AfterQF = nil
}

func (t *SqlFunc[T]) Copy(dest *SqlFunc[T]) {
	dest.Ty = t.Ty
	dest.Ctx = t.Ctx
	dest.Sql = t.Sql
	dest.Args = append(dest.Args[:0], t.Args...)
	dest.SkipSqlErr = t.SkipSqlErr
	dest.BeforeF = t.BeforeF
	dest.AfterEF = t.AfterEF
	dest.AfterQF = t.AfterQF
}

type TxPool[T any] struct {
	p  pool.PoolBlockI[SqlTx[T]]
	db *sql.DB
}

func NewTxPool[T any](db *sql.DB) *TxPool[T] {
	return &TxPool[T]{pool.NewPoolBlock[SqlTx[T]](), db}
}

func (t *TxPool[T]) BeginTx(ctx context.Context, opts ...*sql.TxOptions) *SqlTx[T] {
	var tx = t.p.Get()
	tx.canTx = t.db
	tx.ctx = ctx
	tx.opts = nil
	tx.sqlFuncs = tx.sqlFuncs[:0]
	tx.fin = false
	tx.pool = weak.Make(t)
	if len(opts) > 0 {
		tx.opts = opts[0]
	}
	return tx
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

func (t *SqlTx[T]) SimpleDo(sql string, args ...any) *SqlTx[T] {
	return t.Do(&SqlFunc[T]{
		Sql:  sql,
		Args: args,
	})
}

func (t *SqlTx[T]) Do(sqlf *SqlFunc[T]) *SqlTx[T] {
	sqlf.Sql = strings.TrimSpace(sqlf.Sql)
	t.RawDo(func(dsqlf *SqlFunc[T]) {
		sqlf.Copy(dsqlf)
	})
	return t
}

func (t *SqlTx[T]) RawDo(sqlF func(sqlf *SqlFunc[T])) *SqlTx[T] {
	ps.AppendPtr(&t.sqlFuncs, func(sqlfP *SqlFunc[T]) {
		sqlfP.Clear()
		sqlF(sqlfP)
	})
	return t
}

// PlaceHolder will replaced by ?
func (t *SqlTx[T]) SimplePlaceHolderA(sql string, queryPtr any) *SqlTx[T] {
	return t.DoPlaceHolder(&SqlFunc[T]{Sql: sql}, queryPtr, PlaceHolderA)
}

// PlaceHolder will replaced by $%d
func (t *SqlTx[T]) SimplePlaceHolderB(sql string, queryPtr any) *SqlTx[T] {
	return t.DoPlaceHolder(&SqlFunc[T]{Sql: sql}, queryPtr, PlaceHolderB)
}

// PlaceHolder will replaced by :%d
func (t *SqlTx[T]) SimplePlaceHolderC(sql string, queryPtr any) *SqlTx[T] {
	return t.DoPlaceHolder(&SqlFunc[T]{Sql: sql}, queryPtr, PlaceHolderC)
}

type ReplaceF func(index int, holder string) (replaceTo string)

var (
	// "?"
	PlaceHolderA ReplaceF = func(index int, holder string) (replaceTo string) {
		return "?"
	}
	// "$%d"
	PlaceHolderB ReplaceF = func(index int, holder string) (replaceTo string) {
		return fmt.Sprintf("$%d", index+1)
	}
	// ":%d"
	PlaceHolderC ReplaceF = func(index int, holder string) (replaceTo string) {
		return fmt.Sprintf(":%d", index+1)
	}
)

type paraSort struct {
	key   string
	val   any
	index int
}

var queryPool = pool.NewPoolBlocks[paraSort]()

func (t *SqlTx[T]) DoPlaceHolder(sqlf *SqlFunc[T], queryPtr any, replaceF ReplaceF) *SqlTx[T] {
	if queryPtr == nil {
		return t.Do(sqlf)
	}

	indexM := (*queryPool.Get())[:0]
	defer queryPool.Put(&indexM)

	if dataR := reflect.ValueOf(queryPtr).Elem(); dataR.Kind() == reflect.Map {
		for it := dataR.MapRange(); it.Next(); {
			replaceS := "{" + it.Key().String() + "}"
			if i := strings.Index(sqlf.Sql, replaceS); i != -1 {
				ps.Append(&indexM, func(t *paraSort) {
					t.key = replaceS
					t.val = it.Value().Interface()
					t.index = i
				})
			}
		}
	} else {
		for i := 0; i < dataR.NumField(); i++ {
			field := dataR.Field(i)
			if field.IsValid() && field.CanSet() {
				replaceS := "{" + dataR.Type().Field(i).Name + "}"
				if i := strings.Index(sqlf.Sql, replaceS); i != -1 {
					ps.Append(&indexM, func(t *paraSort) {
						t.key = replaceS
						t.val = field.Interface()
						t.index = i
					})
				}
			}
		}
	}
	if len(indexM) > 1 {
		slices.SortFunc(indexM, func(a, b paraSort) int {
			return a.index - b.index
		})
	}
	sqlf.Args = sqlf.Args[:0]
	for k, v := range ps.Range(indexM) {
		sqlf.Sql = strings.ReplaceAll(sqlf.Sql, v.key, replaceF(k, v.key))
		sqlf.Args = append(sqlf.Args, v.val)
	}
	return t.Do(sqlf)
}

// Deprecated: use sqlFuncs.BeforeF
func (t *SqlTx[T]) BeforeF(f BeforeF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].BeforeF = f
	}
	return t
}

// Deprecated: use sqlFuncs.AfterEF
func (t *SqlTx[T]) AfterEF(f AfterEF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 && t.sqlFuncs[len(t.sqlFuncs)-1].Ty == Execf {
		t.sqlFuncs[len(t.sqlFuncs)-1].AfterEF = f
	}
	return t
}

// Deprecated: use sqlFuncs.AfterQF
func (t *SqlTx[T]) AfterQF(f AfterQF[T]) *SqlTx[T] {
	if len(t.sqlFuncs) > 0 && t.sqlFuncs[len(t.sqlFuncs)-1].Ty == Queryf {
		t.sqlFuncs[len(t.sqlFuncs)-1].AfterQF = f
	}
	return t
}

var (
	ErrTypNil      = errors.New("ErrTypNil")
	ErrBeginTx     = errors.New("ErrBeginTx")
	ErrBeforeF     = errors.New("ErrBeforeF")
	ErrExec        = errors.New("ErrExec")
	ErrAfterExec   = errors.New("ErrAfterExec")
	ErrQuery       = errors.New("ErrQuery")
	ErrAfterQuery  = errors.New("ErrAfterQuery")
	ErrRollback    = errors.New("ErrRollback")
	ErrCommit      = errors.New("ErrCommit")
	ErrHadFin      = errors.New("ErrHadFin")
	ErrUndefinedTy = errors.New("ErrUndefinedTy")
)

type ErrTx[T any] struct {
	Raw    *SqlFunc[T]
	prePtr any
	Typ    error
	Err    error
}

var _ error = &ErrTx[any]{}

// Typ must not nil
func NewErrTx[T any](preErrTx error, Raw *SqlFunc[T], Typ, Err error) (n *ErrTx[T]) {
	if Typ == nil {
		panic(ErrTypNil)
	} else {
		n = &ErrTx[T]{
			Raw: Raw,
			Typ: Typ,
			Err: Err,
		}
		if pre, ok := preErrTx.(*ErrTx[T]); ok && pre != nil {
			n.prePtr = pre
		}
	}
	return
}
func ParseErrTx[T any](err error) *SqlFunc[T] {
	if e, ok := err.(*ErrTx[T]); ok && e != nil {
		return e.Raw
	} else {
		return nil
	}
}
func (t *ErrTx[T]) Is(e error) bool {
	return t.Typ == e || t.Err == e
}
func (t *ErrTx[T]) Error() (s string) {
	var buf strings.Builder
	if t.prePtr != nil {
		buf.WriteString(t.prePtr.(*ErrTx[T]).Error() + "\n")
	}
	if t.Raw != nil {
		buf.WriteString(t.Raw.Sql + "\n")
	}
	if t.Typ != nil {
		buf.WriteString(t.Typ.Error())
	}
	if t.Err != nil {
		buf.WriteString(" > " + t.Err.Error())
	}
	return buf.String()
}

func (t *SqlTx[T]) Fin() (ctxVP T, errTx error) {
	defer func() {
		if txp := t.pool.Value(); txp != nil {
			txp.p.Put(t)
		}
	}()

	if t.fin {
		errTx = NewErrTx[T](errTx, nil, ErrHadFin, nil)
		return
	}

	tx, err := t.canTx.BeginTx(t.ctx, t.opts)
	if err != nil {
		errTx = NewErrTx[T](errTx, nil, ErrBeginTx, err)
		return
	} else {
		var err error
		for _, sqlf := range t.sqlFuncs {
			if sqlf.BeforeF != nil {
				sqlf.BeforeF(&ctxVP, sqlf, &err)
				if err != nil {
					errTx = NewErrTx(errTx, sqlf, ErrBeforeF, err)
					break
				}
			}

			if sqlf.Sql == "" {
				continue
			}

			if sqlf.Ctx == nil {
				sqlf.Ctx = t.ctx
			}

			if sqlf.Ty == null {
				sqlf.Ty = Execf
				if uquery := strings.ToUpper(strings.TrimSpace(sqlf.Sql)); strings.HasPrefix(uquery, "SELECT") {
					sqlf.Ty = Queryf
				}
			}

			if sqlf.Ty == Execf {
				if res, err := tx.ExecContext(sqlf.Ctx, sqlf.Sql, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						errTx = NewErrTx(errTx, sqlf, ErrExec, err)
						break
					}
				} else if sqlf.AfterEF != nil {
					sqlf.AfterEF(&ctxVP, res, &err)
					if err != nil {
						errTx = NewErrTx(errTx, sqlf, ErrAfterExec, err)
						break
					}
				}
			} else if sqlf.Ty == Queryf {
				if res, err := tx.QueryContext(sqlf.Ctx, sqlf.Sql, sqlf.Args...); err != nil {
					if !sqlf.SkipSqlErr {
						errTx = NewErrTx(errTx, sqlf, ErrQuery, err)
						break
					}
				} else if sqlf.AfterQF != nil {
					sqlf.AfterQF(&ctxVP, res, &err)
					if err != nil {
						errTx = NewErrTx(errTx, sqlf, ErrAfterQuery, err)
						break
					}
				}
			} else {
				errTx = NewErrTx(errTx, sqlf, ErrUndefinedTy, nil)
				break
			}
		}
	}
	if errTx != nil {
		if tx != nil {
			if err := tx.Rollback(); err != nil {
				errTx = NewErrTx[T](errTx, nil, ErrRollback, err)
				return
			}
		}
	} else {
		if err := tx.Commit(); err != nil {
			errTx = NewErrTx[T](errTx, nil, ErrCommit, err)
			return
		}
	}
	t.fin = true
	return
}

func IsFin[T any](t *SqlTx[T]) bool {
	return t == nil || t.fin
}

// 复用结构，当前值只能在迭代中使用
type Row[T any] struct {
	Index int
	Raw   T
	Err   error
}

func DealRows[T any](rows *sql.Rows) ([]T, error) {
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

		var (
			stu      = *new(T)
			refV     = reflect.ValueOf(&stu).Elem()
			refT     = reflect.TypeOf(&stu).Elem()
			FieldMap = make(map[string]*reflect.Value)
		)

		for NumField := refV.NumField() - 1; NumField >= 0; NumField-- {
			field := refV.Field(NumField)
			fieldT := refT.Field(NumField)
			fieldTName := fieldT.Name
			if value, ok := fieldT.Tag.Lookup("sql"); ok {
				fieldTName = value
			}
			if !field.IsValid() {
				continue
			}
			if !field.CanSet() {
				FieldMap[strings.ToUpper(fieldTName)] = nil
				continue
			}
			FieldMap[strings.ToUpper(fieldTName)] = &field
		}

		for i := 0; i < len(rowNames); i++ {
			if field, ok := FieldMap[strings.ToUpper(rowNames[i])]; ok {
				if field == nil {
					return nil, fmt.Errorf("DealRows:%s.%s CanSet:false", refT.Name(), rowNames[i])
				}
				val := reflect.ValueOf(*rowP[i].(*any))
				if reflect.TypeOf(*rowP[i].(*any)).ConvertibleTo(field.Type()) {
					field.Set(val)
				} else {
					return nil, fmt.Errorf("DealRows:KindNotMatch:[sql] %v !> [%s.%s] %v", val.Kind(), refT.Name(), rowNames[i], field.Type())
				}
			}
		}
		res = append(res, stu)
	}

	return res, nil
}

func DealRowsIter[T any](rows *sql.Rows) iter.Seq[*Row[T]] {
	var (
		index = 0
		r     = &Row[T]{}
	)
	return func(yield func(*Row[T]) bool) {
		rowNames, err := rows.Columns()
		if err != nil {
			r.Err = err
			if !yield(r) {
				return
			}
		}

		for rows.Next() {
			rowP := make([]any, len(rowNames))
			for i := 0; i < len(rowNames); i++ {
				rowP[i] = new(any)
			}

			err = rows.Scan(rowP...)
			if err != nil {
				r.Err = err
				if !yield(r) {
					break
				}
			}

			var (
				stu      = *new(T)
				refV     = reflect.ValueOf(&stu).Elem()
				refT     = reflect.TypeOf(&stu).Elem()
				FieldMap = make(map[string]*reflect.Value)
			)

			for NumField := refV.NumField() - 1; NumField >= 0; NumField-- {
				field := refV.Field(NumField)
				fieldT := refT.Field(NumField)
				fieldTName := fieldT.Name
				if value, ok := fieldT.Tag.Lookup("sql"); ok {
					fieldTName = value
				}
				if !field.IsValid() {
					continue
				}
				if !field.CanSet() {
					FieldMap[strings.ToUpper(fieldTName)] = nil
					continue
				}
				FieldMap[strings.ToUpper(fieldTName)] = &field
			}

			for i := 0; i < len(rowNames); i++ {
				if field, ok := FieldMap[strings.ToUpper(rowNames[i])]; ok {
					if field == nil {
						r.Err = fmt.Errorf("DealRows:%s.%s CanSet:false", refT.Name(), rowNames[i])
						if !yield(r) {
							return
						}
					}

					val := reflect.ValueOf(*rowP[i].(*any))
					if typ := reflect.TypeOf(*rowP[i].(*any)); typ == nil {
						continue
					} else if typ.ConvertibleTo(field.Type()) {
						field.Set(val)
					} else {
						r.Err = fmt.Errorf("DealRows:KindNotMatch:[sql] %v !> [%s.%s] %v", val.Kind(), refT.Name(), rowNames[i], field.Type())
						if !yield(r) {
							return
						}
					}
				}
			}

			index += 1
			r.Index = index
			r.Err = nil
			r.Raw = stu

			if !yield(r) {
				return
			}
		}
	}
}

func DealRowsMapIter(rows *sql.Rows, caseSwitchF ...CaseSwitchF) iter.Seq[*Row[map[string]any]] {
	var (
		index = 0
		r     = &Row[map[string]any]{}
	)
	return func(yield func(*Row[map[string]any]) bool) {
		rowM := make(map[string]any)

		rowNames, err := rows.Columns()
		if err != nil {
			r.Err = err
			if !yield(r) {
				return
			}
		}

		rowP := make([]any, len(rowNames))
		for i := 0; i < len(rowNames); i++ {
			rowP[i] = new(any)
		}

		for rows.Next() {
			clear(rowM)

			err = rows.Scan(rowP...)
			if err != nil {
				r.Err = err
				if !yield(r) {
					return
				}
			}

			for i := 0; i < len(rowNames); i++ {
				if typ := reflect.TypeOf(*rowP[i].(*any)); typ == nil {
					continue
				}
				key := rowNames[i]
				if len(caseSwitchF) > 0 {
					key = caseSwitchF[0](key)
				}
				rowM[key] = reflect.ValueOf(*rowP[i].(*any)).Interface()
			}
			r.Index = index
			r.Err = nil
			r.Raw = rowM
			index += 1
			if !yield(r) {
				return
			}
		}
	}
}

type CaseSwitchF func(string) string

var ToCamel CaseSwitchF = func(s string) string {
	_count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			_count += 1
		}
	}

	var (
		b    strings.Builder
		has_ bool
	)
	b.Grow(len(s) - _count)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			has_ = true
		} else {
			if c >= utf8.RuneSelf {
				// noASCII
			} else if 'a' <= c && c <= 'z' && has_ {
				// a->A
				c -= 'a' - 'A'
			} else if 'A' <= c && c <= 'Z' && !has_ {
				// A->a
				c += 'a' - 'A'
			}
			b.WriteByte(c)
			has_ = false
		}
	}
	return b.String()
}
