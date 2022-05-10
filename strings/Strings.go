package part

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"time"
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var bb bytes.Buffer
	bb.Grow(leng)
	for i := 0; i < leng; i++ {
		bb.WriteRune(Letters[r.Intn(LettersL)])
	}
	return bb.String()

}

func UnescapeUnicode(raw string) (string, error) {
	return strconv.Unquote(strings.Replace(strconv.Quote(raw), `\\u`, `\u`, -1))
}
