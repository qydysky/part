package v2

import (
	"errors"
	"fmt"
	"go/build"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"unsafe"
)

type Error struct {
	// err       error
	fieldName string
	raw       string
	point     uintptr
	actRaw    string
	actPoint  uintptr
}

// Raw will alloc new Error, which field raw is replaced, but keep other fields.
//
// so errors.Is or Method.IsAction will work fine, but == not
func (t Error) Raw(raw string) Error {
	t.raw = raw
	return t
}

//	func (t *Error) Wrap(err error) *Error {
//		t.err = err
//		return t
//	}
//
//	func (t Error) Unwrap() error {
//		return t.err
//	}
func (t Error) Is(e error) bool {
	if re, ok := e.(Error); ok {
		return re.actPoint == t.actPoint && re.point == t.point
	}
	return false
}
func (t Error) Error() (e string) {
	e = t.fieldName
	// if t.err != nil {
	// 	e += ":" + t.err.Error()
	// }
	if t.raw != "" {
		e += ":" + t.raw
	}
	return
}

// T is a struct which have Fields with type Error or Method,eg.
//
//	xxx := Action[struct{
//		A Error
//		b Error
//		m Method
//	}](`xxx`)
//
// func will use reflect to init these Fields
//
// then use like xxx.A as normal error
func Action[T any](actName string) (act *T) {
	act = new(T)
	for s, v := range reflect.ValueOf(act).Elem().Fields() {
		switch s.Type {
		case reflect.TypeFor[Error]():
			if tagErr := s.Tag.Get(`err`); tagErr != "" {
				reflect.NewAt(reflect.TypeFor[string](), unsafe.Pointer(v.FieldByName("fieldName").UnsafeAddr())).Elem().Set(reflect.ValueOf(tagErr))
			} else {
				reflect.NewAt(reflect.TypeFor[string](), unsafe.Pointer(v.FieldByName("fieldName").UnsafeAddr())).Elem().Set(reflect.ValueOf(s.Name))
			}
			reflect.NewAt(reflect.TypeFor[string](), unsafe.Pointer(v.FieldByName("actRaw").UnsafeAddr())).Elem().Set(reflect.ValueOf(actName))
			reflect.NewAt(reflect.TypeFor[uintptr](), unsafe.Pointer(v.FieldByName("actPoint").UnsafeAddr())).Elem().Set(reflect.ValueOf(uintptr(reflect.ValueOf(act).UnsafePointer())))
			reflect.NewAt(reflect.TypeFor[uintptr](), unsafe.Pointer(v.FieldByName("point").UnsafeAddr())).Elem().Set(reflect.ValueOf(v.UnsafeAddr()))
		case reflect.TypeFor[Method]():
			if pc, file, line, ok := runtime.Caller(1); ok && !strings.HasPrefix(file, build.Default.GOROOT) {
				reflect.NewAt(reflect.TypeFor[string](), unsafe.Pointer(v.FieldByName("actLoc").UnsafeAddr())).Elem().Set(reflect.ValueOf(fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)))
			}
			reflect.NewAt(reflect.TypeFor[string](), unsafe.Pointer(v.FieldByName("actRaw").UnsafeAddr())).Elem().Set(reflect.ValueOf(actName))
			reflect.NewAt(reflect.TypeFor[uintptr](), unsafe.Pointer(v.FieldByName("actPoint").UnsafeAddr())).Elem().Set(reflect.ValueOf(uintptr(reflect.ValueOf(act).UnsafePointer())))
		}
	}
	return
}

type Method struct {
	actLoc   string
	actRaw   string
	actPoint uintptr
}

func (t *Method) Info() (raw, loc string) {
	return t.actRaw, t.actLoc
}

func (t *Method) InAction(err error) bool {
	for {
		switch x := err.(type) {
		case Error:
			return x.actPoint == t.actPoint
		case *Error:
			return x.actPoint == t.actPoint
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			return slices.ContainsFunc(x.Unwrap(), t.InAction)
		default:
			return false
		}
	}
}

// 按格式显示err
//
//	默认使用ErrSimplifyFunc 即 e.Error() + "\n"
func ErrorFormat(e error, format ...ErrFormatFunc) (s string) {
	if e == nil {
		return ""
	}

	if se, ok := e.(interface{ Unwrap() []error }); ok {
		es := se.Unwrap()
		for i := 0; i < len(es); i++ {
			if len(format) > 0 {
				s += format[0](es[i])
			} else {
				s += es[i].Error() + "\n"
			}
		}
	} else if len(format) > 0 {
		s += format[0](e)
	} else {
		s += e.Error()
	}

	return
}

type ErrFormatFunc func(e error) string

var (
	// e.Error() + "\n"
	ErrSimplifyFunc ErrFormatFunc = func(e error) string {
		return e.Error() + "\n"
	}
	// 如是action，则 string(err.action) + ":" + e.Error() + "\n"
	//
	// 否则e.Error() + "\n"
	ErrActionSimplifyFunc ErrFormatFunc = func(e error) string {
		switch x := e.(type) {
		case Error:
			return string(x.actRaw) + ":" + e.Error() + "\n"
		case *Error:
			return string(x.actRaw) + ":" + e.Error() + "\n"
		default:
			return e.Error() + "\n"
		}
	}
	// "> " + e.Error() + " "
	ErrInLineFunc ErrFormatFunc = func(e error) string {
		return "> " + e.Error() + " "
	}
	// 如是action，则 "> " + string(err.action) + ":" + e.Error() + " "
	//
	// 否则"> " + e.Error() + " "
	ErrActionInLineFunc ErrFormatFunc = func(e error) string {
		switch x := e.(type) {
		case Error:
			return fmt.Sprintf("> %v:%v ", x.actRaw, strings.TrimSpace(e.Error()))
		case *Error:
			return fmt.Sprintf("> %v:%v ", x.actRaw, strings.TrimSpace(e.Error()))
		default:
			return fmt.Sprintf("> %v ", strings.TrimSpace(e.Error()))
		}
	}
)

var (
	Is     = errors.Is
	Join   = errors.Join
	Unwrap = errors.Unwrap
)
