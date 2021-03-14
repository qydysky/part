package part

import (
	"fmt"
	"strconv"
)

func Uintptr2String(ptr uintptr) string {
	return fmt.Sprint(ptr)
}


func String2Uintptr(s string) (ptr uintptr) {
	if i,e := strconv.Atoi(s);e != nil {
		return
	} else {
		ptr = uintptr(i)
	}
	return
}