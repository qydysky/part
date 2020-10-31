package part

import (
	"testing"
	p "github.com/qydysky/part"
)

func Test_get(t *testing.T) {
	g := Get(p.Rval{
		Url:"https://www.baidu.com/",
	})
	g.S(`<head><meta http-equiv="`, `"`, 0, 0)
	if g.Err != nil || g.RS != `Content-Type` {return}
	g.S(`<head><meta http-equiv="`, `<meta content="`, 0, 0)
	if g.Err != nil {return}
	if s,e := SS(g.RS, `content="`, `"`, 0, 0);e != nil || s != `text/html;charset=utf-8` {return}
}
