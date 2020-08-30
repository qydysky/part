package part

import (
	goRand "math/rand"
	goCRand "crypto/rand"
  	"math/big"
	"time"
)

type random struct{
	RV []interface{}
}

func Rand() *random {
	return &random{}
}

func (*random) TrueRandom(max int64) int64 {
	var e error
	if r,e := goCRand.Int(goCRand.Reader, big.NewInt(max)); e == nil {
		return r.Int64()
	}
	Logf().E(e.Error())
	return -1
}

func (*random) FakeRandom(max int64) int64 {
	r := goRand.New(goRand.NewSource(time.Now().UnixNano()))
	return r.Int63n(max)
}

func (t *random) MixRandom(max int64) int64 {
	r := t.TrueRandom(max)
	if r != -1 {return r}
	return t.FakeRandom(max)
}