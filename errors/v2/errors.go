package v2

import (
	"errors"
	"fmt"
	"go/build"
	"reflect"
	"runtime"
	"slices"
	"strings"

	us "github.com/qydysky/part/unsafe"
)

type Error struct {
	fieldName string  // Error name, such as errors.New(fieldName)
	actRaw    string  // Action actName
	actPoint  uintptr // Action pointer
	point     uintptr // Error pointer
	info      string  // Error info
}

// WithInfo will alloc new Error, which field info is replaced, but keep other fields.
//
// Error() will return fieldName:info
//
// so errors.Is or Method.IsAction will work fine, but == not
func (t Error) WithInfo(raw string) Error {
	buf := make([]byte, len(t.fieldName)+1+len(raw))
	copy(buf, us.S2B(t.fieldName))
	if raw != "" {
		copy(buf[len(t.fieldName):], []byte{':'})
		copy(buf[len(t.fieldName)+1:], us.S2B(raw))
	}
	t.info = us.B2S(buf)
	return t
}

func (t Error) Is(e error) bool {
	if re, ok := e.(Error); ok {
		return re.actPoint == t.actPoint && re.point == t.point
	}
	return false
}

func (t Error) Error() (e string) {
	return t.info
}

// T is a struct which have Fields with type Error or Method,eg.
//
//	xxx := Action[struct{
//		A Error // Error() return A
//		b Error `err:"a"` // Error() return a
//		c Error `err:"a" info:"b"` // Error() return a:b
//		m Method // some useful func of Action
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
			var name, info string
			if tmp := s.Tag.Get(`err`); tmp != `` {
				name = tmp
			} else {
				name = s.Name
			}
			if tmp := s.Tag.Get(`info`); tmp != `` {
				info = name + `:` + tmp
			} else {
				info = name
			}
			us.SetField(v, 0, name)
			us.SetField(v, 4, info)
			us.SetField(v, 1, actName)
			us.SetField(v, 2, uintptr(reflect.ValueOf(act).UnsafePointer()))
			us.SetField(v, 3, v.UnsafeAddr())
		case reflect.TypeFor[Method]():
			if pc, file, line, ok := runtime.Caller(0); ok && !strings.HasPrefix(file, build.Default.GOROOT) {
				us.SetField(v, 0, fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line))
			}
			us.SetField(v, 1, actName)
			us.SetField(v, 2, uintptr(reflect.ValueOf(act).UnsafePointer()))
		}
	}
	return
}

type Method struct {
	actLoc   string  // Action alloc loc in code
	actRaw   string  // Action actName when init
	actPoint uintptr // Action pointer
}

// Action info
func (t *Method) Info() (raw, loc string) {
	return t.actRaw, t.actLoc
}

// whether err is in Action
func (t *Method) InAction(err error) bool {
	for {
		switch x := err.(type) {
		case Error:
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
