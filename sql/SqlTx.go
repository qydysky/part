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

	pool "github.com/qydysky/part/pool"
	ps "github.com/qydysky/part/slice"
)

type Type int

const (
	null Type = iota
	Execf
	Queryf
)

type CanTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type SqlTx struct {
	canTx    CanTx
	ctx      context.Context
	tx       *sql.Tx
	opts     *sql.TxOptions
	sqlFuncs []*SqlFunc
	fin      bool
	finFunc  func()
}

func BeginTx(canTx CanTx, ctx context.Context, opts ...*sql.TxOptions) *SqlTx {
	var tx = SqlTx{
		canTx: canTx,
		ctx:   ctx,
		tx:    nil,
	}
	if len(opts) > 0 {
		tx.opts = opts[0]
	}
	return &tx
}

func (t *SqlTx) SimpleDo(sql string, args ...any) *SqlTx {
	dsqlf := ps.AppendPtr(&t.sqlFuncs).Clear()
	dsqlf.Sql = strings.TrimSpace(sql)
	dsqlf.Args = args
	dsqlf.autoType()
	return t
}

func (t *SqlTx) Do(sqlf *SqlFunc) *SqlTx {
	sqlf.Sql = strings.TrimSpace(sqlf.Sql)
	sqlf.autoType()
	sqlf.Copy(ps.AppendPtr(&t.sqlFuncs).Clear())
	return t
}

func (t *SqlTx) StopWithErr(e error) *SqlTx {
	ps.AppendPtr(&t.sqlFuncs).Clear().BeforeF = func(sqlf *SqlFunc) error {
		return e
	}
	return t
}

// PlaceHolder will replaced by ?
func (t *SqlTx) SimplePlaceHolderA(sql string, queryPtr any) *SqlTx {
	return t.DoPlaceHolder(&SqlFunc{Sql: sql}, queryPtr, PlaceHolderA)
}

// PlaceHolder will replaced by $%d
func (t *SqlTx) SimplePlaceHolderB(sql string, queryPtr any) *SqlTx {
	return t.DoPlaceHolder(&SqlFunc{Sql: sql}, queryPtr, PlaceHolderB)
}

// PlaceHolder will replaced by :%d
func (t *SqlTx) SimplePlaceHolderC(sql string, queryPtr any) *SqlTx {
	return t.DoPlaceHolder(&SqlFunc{Sql: sql}, queryPtr, PlaceHolderC)
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
	key    string
	val    any
	index  int
	valLen int
}

var queryPool = pool.NewPoolBlocks[paraSort]()

func (t *SqlTx) DoPlaceHolder(sqlf *SqlFunc, queryPtr any, replaceF ReplaceF) *SqlTx {
	// sqlf.Sql = strings.TrimSpace(sqlf.Sql)
	sqlf = sqlf.Copy(ps.AppendPtr(&t.sqlFuncs).Clear())
	defer sqlf.autoType()

	if queryPtr == nil {
		return t
	}

	indexM := queryPool.Get()
	defer queryPool.Put(indexM)

	*indexM = (*indexM)[:0]

	if dataR := reflect.ValueOf(queryPtr).Elem(); dataR.Kind() == reflect.Map {
		for it := dataR.MapRange(); it.Next(); {
			replaceS := "{" + it.Key().String() + "}"
			if i := strings.Index(sqlf.Sql, replaceS); i != -1 {
				t := ps.Append(indexM)
				t.key = replaceS
				t.val = it.Value().Interface()
				switch it.Value().Kind() {
				case reflect.Slice, reflect.Array:
					t.valLen = it.Value().Len()
				default:
				}
				t.index = i
			}
		}
	} else {
		for i := 0; i < dataR.NumField(); i++ {
			field := dataR.Field(i)
			if field.IsValid() && field.CanSet() {
				replaceS := "{" + dataR.Type().Field(i).Name + "}"
				if i := strings.Index(sqlf.Sql, replaceS); i != -1 {
					t := ps.Append(indexM)
					t.key = replaceS
					t.val = field.Interface()
					switch field.Kind() {
					case reflect.Slice, reflect.Array:
						t.valLen = field.Len()
					default:
					}
					t.index = i
				}
			}
		}
	}
	if len(*indexM) > 1 {
		slices.SortFunc(*indexM, func(a, b paraSort) int {
			return a.index - b.index
		})
	}
	sqlf.Args = sqlf.Args[:0]
	for k, v := range ps.Range(*indexM) {
		if v.valLen > 0 {
			var (
				replaceS string
				item     = replaceF(k, v.key)
			)
			for i := 0; i < v.valLen; i++ {
				replaceS += item
				if i < v.valLen-1 {
					replaceS += `,`
				}
			}
			sqlf.Sql = strings.ReplaceAll(sqlf.Sql, v.key, replaceS)
			for i := range toAnySlice(v.val) {
				sqlf.Args = append(sqlf.Args, i)
			}
		} else {
			sqlf.Sql = strings.ReplaceAll(sqlf.Sql, v.key, replaceF(k, v.key))
			sqlf.Args = append(sqlf.Args, v.val)
		}
	}
	return t
}

func (t *SqlTx) BeforeF(f BeforeF) *SqlTx {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].BeforeF = f
	}
	return t
}

func (t *SqlTx) AfterEF(f AfterEF) *SqlTx {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].AfterEF = f
	}
	return t
}

func (t *SqlTx) AfterQF(f AfterQF) *SqlTx {
	if len(t.sqlFuncs) > 0 {
		t.sqlFuncs[len(t.sqlFuncs)-1].AfterQF = f
	}
	return t
}

func (t *SqlTx) FinF(f func()) *SqlTx {
	t.finFunc = f
	return t
}

func (t *SqlTx) Run() (errTx error) {
	return t.commitOrRollback(t.do())
}

func (t *SqlTx) AddToTxs(txs *SqlTxs) {
	txs.AddTx(t)
}

// must call commitOrRollback
func (t *SqlTx) do() (errTx error) {
	if t.fin {
		panic(ErrHadFin)
	}

	if tx, err := t.canTx.BeginTx(t.ctx, t.opts); err != nil {
		errTx = NewErrTx(errTx, ErrBeginTx, err)
		return
	} else {
		t.tx = tx
	}

	for _, sqlf := range t.sqlFuncs {
		if sqlf.BeforeF != nil {
			if err := sqlf.BeforeF(sqlf); err != nil {
				errTx = NewErrTx(errTx, ErrBeforeF, err).WithRaw(sqlf)
				break
			}
		}

		if sqlf.Sql == "" {
			continue
		}

		if sqlf.Ctx == nil {
			sqlf.Ctx = t.ctx
		}

		if sqlf.Ty == Execf {
			if res, err := t.tx.ExecContext(sqlf.Ctx, sqlf.Sql, sqlf.Args...); err != nil {
				errTx = NewErrTx(errTx, ErrExec, err).WithRaw(sqlf)
				break
			} else if sqlf.AfterEF != nil {
				if err := sqlf.AfterEF(res); err != nil {
					errTx = NewErrTx(errTx, ErrAfterExec, err).WithRaw(sqlf)
					break
				}
			}
		} else if sqlf.Ty == Queryf {
			if res, err := t.tx.QueryContext(sqlf.Ctx, sqlf.Sql, sqlf.Args...); err != nil {
				errTx = NewErrTx(errTx, ErrQuery, err).WithRaw(sqlf)
				break
			} else {
				if sqlf.AfterQF != nil {
					if err := sqlf.AfterQF(res); err != nil {
						res.Close()
						errTx = NewErrTx(errTx, ErrAfterQuery, err).WithRaw(sqlf)
						break
					}
				}
				res.Close()
			}
		} else {
			errTx = NewErrTx(errTx, ErrUndefinedTy, nil).WithRaw(sqlf)
			break
		}
	}

	return
}

func (t *SqlTx) commitOrRollback(errTx error) error {
	if t.tx != nil {
		if errTx != nil {
			if !HasErrTx(errTx, ErrBeginTx, ErrCommit, ErrRollback) {
				if err := t.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
					errTx = NewErrTx(errTx, ErrRollback, err)
				}
			}
		} else {
			if err := t.tx.Commit(); err != nil {
				errTx = NewErrTx(errTx, ErrCommit, err)
			}
		}
	}
	t.tx = nil
	t.fin = true
	if t.finFunc != nil {
		t.finFunc()
	}
	return errTx
}

func toAnySlice(t any) iter.Seq[any] {
	return func(yield func(any) bool) {
		switch a := t.(type) {
		case []string:
			rangeV(a, yield)
		case []uint:
			rangeV(a, yield)
		case []uint32:
			rangeV(a, yield)
		case []uint64:
			rangeV(a, yield)
		case []int:
			rangeV(a, yield)
		case []int32:
			rangeV(a, yield)
		case []int64:
			rangeV(a, yield)
		case []float32:
			rangeV(a, yield)
		case []float64:
			rangeV(a, yield)
		case []any:
			rangeV(a, yield)
		default:
			if !yield(a) {
				return
			}
		}
	}
}

func rangeV[T any](a []T, yield func(any) bool) {
	for i := range a {
		if !yield(a[i]) {
			return
		}
	}
}
