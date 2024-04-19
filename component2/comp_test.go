package component2

import (
	"testing"
)

type B struct{}

func (b B) AddOne(a int) int {
	return a + 1
}

func init() {
	if e := Register[a]("github.com/qydysky/part/component2.aa", B{}); e != nil {
		panic(e)
	}
	aa = Get[a](pkgid)
}

type a interface {
	AddOne(int) int
}

// or var aa = Get[a](pkgid)
var aa a

var pkgid = PkgId("aa")

func Test(t *testing.T) {
	if aa.AddOne(1) != 2 {
		t.Fatal()
	}
}
