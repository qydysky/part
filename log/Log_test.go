package part

import (
	// "fmt"

	"testing"
	"time"

	"net/http"
	_ "net/http/pprof"
)

func Test_1(t *testing.T) {
	n := New(Config{
		File:          `1.log`,
		Stdout:        true,
		Prefix_string: map[string]struct{}{`T:`: On, `I:`: On, `W:`: On, `E:`: On},
	})

	n.L(`T:`, `s`).L(`I:`, `s`).Block(1000)
	n.Log_to_file(`2.log`).L(`W:`, `s`).L(`E:`, `s`)

	{
		n1 := n.Base(`>1`)
		n1.L(`T:`, `s`).L(`I:`, `s`)
		{
			n2 := n1.Base_add(`>2`)
			n2.L(`T:`, `s`).L(`I:`, `s`)
		}
	}

	n.Level(map[string]struct{}{`W:`: On}).L(`T:`, `s`).L(`I:`, `s`).L(`W:`, `s`).L(`E:`, `s`)
	n.Block(1000)
}

var n *Log_interface

func Test_2(t *testing.T) {
	n = New(Config{
		File:          `1.log`,
		Stdout:        true,
		Prefix_string: map[string]struct{}{`T:`: On, `I:`: On, `W:`: On, `E:`: On},
	})

	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()
	// n = nil
	for {
		n := n.Base_add(`>1`)
		n.L(`T:`, `s`)
		time.Sleep(time.Second * time.Duration(1))
		// n=nil
	}
}
