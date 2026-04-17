package unsafe

import (
	"reflect"
	"unsafe"
)

func SetField[T, B any, C string | int](strutPoint T, fieldNameOrIndex C, val B) {
	switch x := any(strutPoint).(type) {
	case reflect.Value:
		if !x.CanAddr() {
			panic("strutPoint CanAddr false")
		} else {
			switch y := any(fieldNameOrIndex).(type) {
			case int:
				x = x.Field(y)
			case string:
				x = x.FieldByName(y)
			}
			if !x.IsValid() {
				panic("field no exist")
			}
			if x.CanSet() {
				x.Set(reflect.ValueOf(val))
			} else {
				reflect.NewAt(reflect.TypeFor[B](), unsafe.Pointer(x.UnsafeAddr())).Elem().Set(reflect.ValueOf(val))
			}
		}
	default:
		if vf := reflect.ValueOf(x); vf.Kind() != reflect.Pointer {
			panic("strutPoint is not pointer")
		} else {
			switch y := any(fieldNameOrIndex).(type) {
			case int:
				vf = vf.Elem().Field(y)
			case string:
				vf = vf.Elem().FieldByName(y)
			}
			if !vf.IsValid() {
				panic("field no exist")
			} else if vf.CanSet() {
				vf.Set(reflect.ValueOf(val))
			} else {
				reflect.NewAt(reflect.TypeFor[B](), unsafe.Pointer(vf.UnsafeAddr())).Elem().Set(reflect.ValueOf(val))
			}
		}
	}
}
