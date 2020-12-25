package part

import (
	"time"
	"testing"
)

type test_item struct {
	data string
}

func Test_1(t *testing.T) {
    n := New(Config{
        Prefix_string:map[string]struct{}{`T:`:On,`I:`:On,`W:`:On,`E:`:On},
    })

    n.Log_to_file(`1.log`).L(`T:`,`s`).L(`I:`,`s`).L(`W:`,`s`).L(`E:`,`s`)

    {
        n1 := n.Base(`>1`)
        n1.L(`T:`,`s`).L(`I:`,`s`)
        {
            n2 := n1.Base_add(`>2`)
            n2.L(`T:`,`s`).L(`I:`,`s`)
        }
    }

    n.Level(map[string]struct{}{`W:`:On}).L(`T:`,`s`).L(`I:`,`s`).L(`W:`,`s`).L(`E:`,`s`)
    n.Block(1000)
}