package part

import (
	"reflect"
	"time"
)

//语法糖

// 数组切片，重新分配内存
// i := []int{0,1,2}
// b := SliceCut(i[:1]).([]int)
func SliceCopy(src interface{}) (des interface{}) {
	srcV := reflect.ValueOf(src)
	if sk := srcV.Kind(); sk != reflect.Slice && sk != reflect.Array {
		panic(&reflect.ValueError{Method: "reflect.Copy", Kind: sk})
	}
	desV := reflect.MakeSlice(srcV.Type(), srcV.Len(), srcV.Len())
	reflect.Copy(desV, srcV)
	des = desV.Interface()
	return
}

func Callback(f func(startT time.Time, args ...any)) func(args ...any) {
	now := time.Now()
	return func(args ...any) {
		f(now, args...)
	}
}
