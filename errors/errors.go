package errors

import (
	"errors"
)

type Error struct {
	son    error
	Reason string
	action string
}

func (t Error) Error() string {
	return t.Reason
}

func (t Error) WithReason(reason string) Error {
	t.Reason = reason
	return t
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

// Grow will append error action for catch
func Grow(fe Error, e error) Error {
	if v, ok := e.(Error); ok {
		fe.son = v
	} else {
		fe.son = Error{
			Reason: e.Error(),
		}
	}
	return fe
}

func New(action string, reason ...string) (e Error) {
	e = Error{
		action: action,
	}
	if len(reason) > 0 {
		e.Reason = reason[0]
	}
	return
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

func ErrorFormat(e error, format ...ErrFormat) (s string) {
	if e == nil {
		return ""
	}

	if se, ok := e.(interface {
		Unwrap() []error
	}); ok {
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

type ErrFormat func(e error) string

var (
	ErrSimplifyFunc = func(e error) string {
		return e.Error() + "\n"
	}
	ErrActionSimplifyFunc = func(e error) string {
		if err, ok := e.(Error); ok {
			return err.action + ":" + e.Error() + "\n"
		} else {
			return e.Error() + "\n"
		}
	}
	ErrInLineFunc = func(e error) string {
		return "> " + e.Error() + " "
	}
	ErrActionInLineFunc = func(e error) string {
		if err, ok := e.(Error); ok {
			return "> " + err.action + ":" + e.Error() + " "
		} else {
			return "> " + e.Error() + " "
		}
	}
)
