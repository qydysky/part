package part

import (
	"bytes"
	"math/rand/v2"
	"strconv"
	"strings"
)

const (
	Number RandType = iota
	Hex
	LowNumber
	UppNumber
)

type RandType int

func Rand(typel RandType, leng int) string {
	source := "0123456789"
	if typel >= Hex {
		source += "abcdef"
	}
	if typel >= LowNumber {
		source += "ghijklmnopqrstuvwxyz"
	}
	if typel >= UppNumber {
		source += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	return Randv2(leng, source)
}

func Randv2(leng int, source string) string {
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
