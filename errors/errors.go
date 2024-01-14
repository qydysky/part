package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	son    interface{}
	reason string
	action string
}

func (t Error) Error() string {
	return t.reason
}

func Catch(e error, action string) bool {
	if v, ok := e.(Error); ok {
		if v.action == action {
			return true
		} else if v.son != nil {
			return Catch((v.son).(Error), action)
		}
	}
	return false
}

func Grow(e error, son Error) Error {
	if v, ok := e.(Error); ok {
		son.son = v
	} else {
		son.son = Error{
			reason: v.Error(),
		}
	}
	return son
}

func New(reason string, action string) Error {
	return Error{
		reason: reason,
		action: action,
	}
}

func Join(e ...error) error {
	if len(e) == 0 {
		return nil
	}

	var errs []error
	for _, v := range e {
		if e, ok := v.(interface {
			Unwrap() []error
		}); ok {
			errs = append(errs, e.Unwrap()...)
		} else {
			errs = append(errs, v)
		}
	}

	return errors.Join(errs...)
}

func Unwrap(e error) []error {
	if e == nil {
		return []error{}
	}

	if e, ok := e.(interface {
		Unwrap() []error
	}); ok {
		return e.Unwrap()
	}

	return []error{errors.Unwrap(e)}
}

func ErrorFormat(e error, format ...func(error) string) (s string) {
	if e == nil {
		return ""
	}

	if se, ok := e.(interface {
		Unwrap() []error
	}); ok {
		for _, v := range se.Unwrap() {
			if len(format) > 0 {
				s += format[0](v)
			} else {
				s += e.Error() + "\n"
			}
		}
	} else if len(format) > 0 {
		s += format[0](e)
	} else {
		s += e.Error()
	}

	return
}

var (
	ErrSimplifyFunc = func(e error) string {
		if es := e.Error(); len(es) > 20 {
			return fmt.Sprintf("%.16s...\n", es)
		} else {
			return es + "\n"
		}
	}
)
