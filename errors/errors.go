package errors

import (
	"errors"
)

type Action string

func (t Action) Append(child string) Action {
	return Action(string(t) + child)
}

func (t Action) New(reason ...string) (e Error) {
	e = Error{
		action: t,
		Reason: string(t),
	}
	if len(reason) > 0 {
		e.Reason = reason[0]
	}
	return
}

func (t Action) Error() string {
	return string(t)
}

func (t Action) Unwrap() []error {
	return []error{
		Error{
			action: t,
			Reason: string(t),
		},
	}
}

func (t Action) Catch(e error) bool {
	return Catch(e, t)
}

type Error struct {
	Reason string
	action Action
}

func (t Error) Is(e error) bool {
	return t.Error() == e.Error()
}

func (t Error) Error() string {
	return t.Reason
}

func (t Error) Unwrap() error {
	return nil
}

func (t Error) WithReason(reason string) Error {
	t.Reason = reason
	return t
}

func Catch(e error, action Action) bool {
	if v, ok := e.(Error); ok {
		if v.action == action {
			return true
		}
	}
	for _, err := range Unwrap(e) {
		if v, ok := err.(Error); ok {
			if v.action == action {
				return true
			}
		}
	}
	return false
}

func New(reason string, action ...Action) (e Error) {
	e = Error{
		Reason: reason,
	}
	if len(action) > 0 {
		e.action = action[0]
	}
	return
}

func Join(e ...error) error {
	if len(e) == 0 {
		return nil
	}

	var errs []error
	for _, v := range e {
		switch x := v.(type) {
		case interface{ Unwrap() []error }:
			errs = append(errs, x.Unwrap()...)
		default:
			errs = append(errs, v)
		}
	}

	return errors.Join(errs...)
}

func Unwrap(e error) []error {
	if e == nil {
		return []error{}
	}

	switch x := e.(type) {
	case interface{ Unwrap() error }:
		return []error{x.Unwrap()}
	case interface{ Unwrap() []error }:
		return x.Unwrap()
	default:
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
	ErrSimplifyFunc ErrFormat = func(e error) string {
		return e.Error() + "\n"
	}
	ErrActionSimplifyFunc ErrFormat = func(e error) string {
		if err, ok := e.(Error); ok && string(err.action) != err.Reason {
			return string(err.action) + ":" + e.Error() + "\n"
		} else {
			return e.Error() + "\n"
		}
	}
	ErrInLineFunc ErrFormat = func(e error) string {
		return "> " + e.Error() + " "
	}
	ErrActionInLineFunc ErrFormat = func(e error) string {
		if err, ok := e.(Error); ok && string(err.action) != err.Reason {
			return "> " + string(err.action) + ":" + e.Error() + " "
		} else {
			return "> " + e.Error() + " "
		}
	}
)
