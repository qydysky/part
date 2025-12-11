//go:build !race

package part

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func Test7(t *testing.T) {
	if rul := testing.Benchmark(Benchmark1); rul.AllocedBytesPerOp() > 1024 || rul.AllocsPerOp() > 22 {
		t.Fatal()
	}
}

// 18697 ns/op            1024 B/op         22 allocs/op
func Benchmark1(b *testing.B) {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	txPool := NewTxPool(db)
	if e := txPool.BeginTx(context.Background()).Do(&SqlFunc{
		Ty:  Execf,
		Sql: "create table log123 (a INTEGER,b DATE,c text)",
	}).Run(); e != nil {
		b.Fatal(e)
	}

	x := txPool.BeginTx(context.Background())
	x.SimplePlaceHolderA("insert into log123 values (1,'2025-01-01',3)", nil)
	if e := x.Run(); e != nil {
		b.Fatal(e)
	}

	sqlF := &SqlFunc{
		Ty:  Queryf,
		Sql: "select 1 from log123 where 1=0",
	}

	for b.Loop() {
		if err := txPool.BeginTx(context.Background()).Do(sqlF).Run(); err != nil {
			b.Fatal(err)
		}
	}
}
