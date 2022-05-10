package part

import (
	"testing"
)

func Test_Rand(t *testing.T) {
	t.Log(Rand(Number, 14))
	t.Log(Rand(lowNumber, 14))
	t.Log(Rand(uppNumber, 14))
}

func Test_UnescapeUnicode(t *testing.T) {
	s, e := UnescapeUnicode("\uB155, \uC138\uC0C1(\u4E16\u4E0A). \u263a")
	t.Log(s)
	t.Log(e)
}
