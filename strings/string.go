package part

import (
	"strconv"
	"strings"
)

func UnescapeUnicode(raw string) (string, error) {
	return strconv.Unquote(strings.Replace(strconv.Quote(raw), `\\u`, `\u`, -1))
}
