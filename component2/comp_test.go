package component2

import (
	"testing"
)

type B struct {
	i int
}

func (b *B) AddOne(any) int {
	return 2
}

func Test5(t *testing.T) {
	if e := Register[interface {
		AddOne(any) int
	}]("aa5", &B{}); e != nil {
		panic(e)
	}

	if UnRegist("aa-5") != ErrNoFound {
		t.Fatal()
	}

	if UnRegist("aa5") != nil {
		t.Fatal()
	}
}

func Test4(t *testing.T) {
	t.Log(PkgId())
}

func Test3(t *testing.T) {
	if e := Register[interface {
		AddOne(any) int
	}]("aa3", &B{}); e != nil {
		panic(e)
	}

	aa := GetV3[interface {
		AddOne(any) int
	}]("aa3")

	aa.Run2(func(inter interface{ AddOne(any) int }) {
		if inter.AddOne(func(i int) int { return i + 1 }) != 2 {
			t.Fatal()
		}
	})
}

func Test1(t *testing.T) {
	if e := Register[interface {
		AddOne(any) int
	}]("aa1", &B{}); e != nil {
		panic(e)
	}

	aa := GetV3[interface {
		AddOne(any) int
	}]("aa-1").Inter(func(err error) interface{ AddOne(any) int } {
		return &B{}
	})

	if aa.AddOne(func(i int) int { return i + 1 }) != 2 {
		t.Fatal()
	}
}

func Test(t *testing.T) {
	if e := Register[interface {
		AddOne(any) int
	}]("aa", &B{}); e != nil {
		panic(e)
	}

	aa := Get[interface {
		AddOne(any) int
	}]("aa")

	if aa.AddOne(func(i int) int { return i + 1 }) != 2 {
		t.Fatal()
	}
}

type C struct {
	F func(a, b int) int
}

func (c C) CallF(a, b int) int {
	return c.F(a, b)
}

func Test2(t *testing.T) {
	type a interface{ CallF(int, int) int }
	if e := Register[a]("aa20", C{func(a, b int) int { return a + b }}); e != nil {
		panic(e)
	}

	{
		GetV2("aa20", PreFuncCu[a]{}).Run(func(s a) error {
			if 3 != s.CallF(1, 2) {
				t.Fatal()
			}
			return nil
		})
	}

	{
		ok := false
		GetV2("aa20", PreFuncCu[a]{
			Initf: func(b a) a {
				ok = true
				return b
			},
		})
		if !ok {
			t.Fatal()
		}
	}

	{
		ok := false
		GetV2("aa21", PreFuncCu[a]{
			ErrNoFoundf: func(id string) error {
				ok = true
				return ErrNoFound
			},
		})
		if !ok {
			t.Fatal()
		}
	}

	{
		ok := false
		GetV2("aa20", PreFuncCu[interface{ Add() }]{
			ErrTypeAssertionf: func(id string) error {
				ok = true
				return ErrTypeAssertion
			},
		})
		if !ok {
			t.Fatal()
		}
	}
}
