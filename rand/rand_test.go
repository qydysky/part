package part

import (
	"bytes"
	"testing"
)

func Test_Rand(t *testing.T) {
	t.Log(Rand[string](TypeNum|TypeLow, 14))
	t.Log(Rand[string](TypeLow, 14))
	t.Log(Rand[string](TypeUpp, 14))
}

func Test2(t *testing.T) {
	r := RandReader(TypeNum|TypeLow|TypeUpp, 100)
	buf := bytes.NewBuffer([]byte{})
	buf.ReadFrom(r)
	t.Log(buf.String())
}
