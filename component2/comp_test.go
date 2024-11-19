package component2

import (
	"testing"
)

type B struct{}

func (b B) AddOne(any) int {
	return 2
}

func Test(t *testing.T) {
	if e := Register[interface {
		AddOne(any) int
	}]("aa", B{}); e != nil {
		panic(e)
	}

	aa := Get[interface {
		AddOne(any) int
	}]("aa")

	if aa.AddOne(func(i int) int { return i + 1 }) != 2 {
		t.Fatal()
	}
}
