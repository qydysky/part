package unsafe

import "unsafe"

func B2S(s []byte) string {
	return unsafe.String(unsafe.SliceData(s), len(s))
}

func S2B(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
