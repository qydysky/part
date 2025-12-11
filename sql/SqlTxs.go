package part

import (
	"errors"

	pool "github.com/qydysky/part/pool"
)

type SqlTxs struct {
	txs []*SqlTx
}

var txsPool = pool.NewPoolBlock[SqlTxs]()

func NewSqlTxs() *SqlTxs {
	return txsPool.Get().clear()
}

func (t *SqlTxs) clear() *SqlTxs {
	t.txs = t.txs[:0]
	return t
}

func (t *SqlTxs) AddTx(tx *SqlTx) *SqlTxs {
	t.txs = append(t.txs, tx)
	return t
}

var errOtherTx = errors.New("errOtherTx")

func (t *SqlTxs) Run() (errTxI int, e error) {
	var cuTx int
	for i := 0; i < len(t.txs) && e == nil; i++ {
		cuTx = i
		e = t.txs[i].do()
	}
	e = t.txs[cuTx].commitOrRollback(e)
	if e != nil {
		errTxI = cuTx
	}
	for cuTx -= 1; cuTx >= 0; cuTx-- {
		if e != nil {
			_ = t.txs[cuTx].commitOrRollback(errOtherTx)
		} else {
			_ = t.txs[cuTx].commitOrRollback(nil)
		}
	}
	return
}
