package part

import (
	"context"
	"database/sql"

	pool "github.com/qydysky/part/pool"
)

type TxPool struct {
	p  pool.PoolBlockI[SqlTx]
	db *sql.DB
}

func NewTxPool(db *sql.DB) *TxPool {
	return &TxPool{pool.NewPoolBlock[SqlTx](), db}
}

func (t *TxPool) BeginTx(ctx context.Context, opts ...*sql.TxOptions) *SqlTx {
	var tx = t.p.Get()
	tx.canTx = t.db
	tx.ctx = ctx
	tx.tx = nil
	tx.opts = nil
	tx.sqlFuncs = tx.sqlFuncs[:0]
	tx.fin = false
	tx.finFunc = func() {
		t.p.Put(tx)
	}
	if len(opts) > 0 {
		tx.opts = opts[0]
	}
	return tx
}
