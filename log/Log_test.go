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
        Level_string:[4]string{`T`,`I`,`W`,`E`},
    })
    time.Sleep(time.Second)

    n.Log_to_file(`1.log`).T(`s`).I("11").W(`12`).E(`a`)

    {
        n1 := n.Base(`>1`)
        n1.T(`s`).I("11")
        {
            n2 := n1.Base_add(`>2`)
            n2.W(`12`).E(`a`)
        }
    }

    n.Level(2).T(`s`).I("11").W(`12`).E(`a`)
    n.Block(1000)
}