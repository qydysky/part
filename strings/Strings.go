package part

import (
	"bytes"
	"math/rand/v2"
	"strconv"
	"strings"
)

const (
	Number    RandType = 0
	LowNumber RandType = 1
	UppNumber RandType = 2
)

type RandType int

func Rand(typel RandType, leng int) string {
	source := "0123456789"
	if typel > 0 {
		source += "abcdefghijklmnopqrstuvwxyz"
	}
	if typel > 1 {
		source += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	Letters := []rune(source)
	LettersL := len(Letters)
	var bb bytes.Buffer
	bb.Grow(leng)
	for i := 0; i < leng; i++ {
		bb.WriteRune(Letters[int(rand.Uint64N(uint64(LettersL)))])
	}
	return bb.String()

}

func UnescapeUnicode(raw string) (string, error) {
	return strconv.Unquote(strings.Replace(strconv.Quote(raw), `\\u`, `\u`, -1))
}
