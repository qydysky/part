package part

import (
	"context"
	"database/sql"
)

type BeforeF func(sqlf *SqlFunc) error
type AfterEF func(result sql.Result) error
type AfterQF func(rows *sql.Rows) error

type SqlFunc struct {
	Ty      Type            // 	Execf or Queryf, default: auto detection
	Ctx     context.Context // default: use transaction Ctx
	Sql     string
	Args    []any
	BeforeF BeforeF
	AfterEF AfterEF
	AfterQF AfterQF
}

func (t *SqlFunc) Clear() *SqlFunc {
	t.Ty = null
	t.Ctx = context.Background()
	t.Sql = ""
	t.Args = t.Args[:0]
	t.BeforeF = nil
	t.AfterEF = nil
	t.AfterQF = nil
	return t
}

func (t *SqlFunc) Copy(dest *SqlFunc) *SqlFunc {
	dest.Ty = t.Ty
	dest.Ctx = t.Ctx
	dest.Sql = t.Sql
	dest.Args = append(dest.Args[:0], t.Args...)
	dest.BeforeF = t.BeforeF
	dest.AfterEF = t.AfterEF
	dest.AfterQF = t.AfterQF
	return dest
}
