package part

import (
	"math/rand"
	"time"
	"bytes"
)

type stringl struct{}

func Stringf() *stringl {
	return &stringl{}
}

func (t *stringl)Rand(typel,leng int) string {
	source := "0123456789"
	if typel > 0 {source+="abcdefghijklmnopqrstuvwxyz"}
	if typel > 1 {source+="ABCDEFGHIJKLMNOPQRSTUVWXYZ"}

	Letters := []rune(source)
	LettersL := len(Letters)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var bb bytes.Buffer
	bb.Grow(leng)
	for i := 0; i < leng; i++ {
		bb.WriteRune(Letters[r.Intn(LettersL)])
	}
	return bb.String()

}
