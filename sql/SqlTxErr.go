package part

import (
	"errors"
	"fmt"
	"go/build"
	"iter"
	"runtime"
	"strings"
)

var (
	ErrTypNil      = errors.New("ErrTypNil")      // panic error
	ErrPreErrWrong = errors.New("ErrPreErrWrong") // panic error
	ErrBeginTx     = errors.New("ErrBeginTx")
	ErrBeforeF     = errors.New("ErrBeforeF")
	ErrExec        = errors.New("ErrExec")
	ErrAfterExec   = errors.New("ErrAfterExec")
	ErrQuery       = errors.New("ErrQuery")
	ErrAfterQuery  = errors.New("ErrAfterQuery")
	ErrRollback    = errors.New("ErrRollback")
	ErrCommit      = errors.New("ErrCommit")
	ErrHadFin      = errors.New("ErrHadFin") // panic error
	ErrUndefinedTy = errors.New("ErrUndefinedTy")
)

type ErrTx struct {
	Raw      *SqlFunc
	prePtr   any
	callTree string
	Typ      error
	Err      error
}

var _ error = &ErrTx{}

// Typ must not nil
func NewErrTx(preErrTx error, Typ, Err error) (n *ErrTx) {
	if Typ == nil {
		panic(ErrTypNil)
	} else {
		n = &ErrTx{
			Typ:      Typ,
			callTree: *getCall(1),
			Err:      Err,
		}
		if preErrTx != nil {
			if pre, ok := preErrTx.(*ErrTx); ok && pre != nil {
				n.prePtr = pre
			} else {
				panic(ErrPreErrWrong)
			}
		}
	}
	return
}
func ParseErrTx(err error) *ErrTx {
	if e, ok := err.(*ErrTx); ok && e != nil {
		return e
	} else {
		return nil
	}
}
func HasErrTx(err any, errs ...error) bool {
	if e, ok := err.(*ErrTx); ok && e != nil {
		for v := range e.ForwardRange() {
			for _, v1 := range errs {
				if v == v1 || errors.Is(v, v1) {
					return true
				}
			}
		}
		return false
	} else if e, ok := err.(error); ok && e != nil {
		for _, v1 := range errs {
			if e == v1 || errors.Is(e, v1) {
				return true
			}
		}
		return false
	} else if err == nil {
		for _, v1 := range errs {
			if v1 == nil {
				return true
			}
		}
		return false
	} else {
		return false
	}
}
func (t *ErrTx) WithRaw(raw *SqlFunc) *ErrTx {
	t.Raw = raw
	return t
}
func (t *ErrTx) ForwardRange() iter.Seq[*ErrTx] {
	return func(yield func(*ErrTx) bool) {
		for tmp := t; tmp != nil; tmp, _ = tmp.prePtr.(*ErrTx) {
			if !yield(tmp) {
				return
			}
		}
	}
}
func (t *ErrTx) Is(e error) bool {
	for tmp := range t.ForwardRange() {
		if tmp.Typ == e || tmp.Err == e {
			return true
		}
	}
	return false
}
func (t *ErrTx) Error() (s string) {
	var buf strings.Builder
	if t.prePtr != nil {
		buf.WriteString(t.prePtr.(*ErrTx).Error() + "\n")
	}
	if t.Raw != nil {
		buf.WriteString(t.Raw.Sql + "\n")
	}
	if t.Typ != nil {
		buf.WriteString(t.Typ.Error())
	}
	if t.Err != nil {
		buf.WriteString(" > " + t.Err.Error())
	}
	if t.callTree != "" {
		buf.WriteString(t.callTree + "\n")
	}
	return buf.String()
}

func getCall(i int) (calls *string) {
	var cs string
	for i += 1; true; i++ {
		if pc, file, line, ok := runtime.Caller(i); !ok || strings.HasPrefix(file, build.Default.GOROOT) {
			break
		} else {
			cs += fmt.Sprintf("\ncall by %s\n\t%s:%d", runtime.FuncForPC(pc).Name(), file, line)
		}
	}
	if cs == "" {
		cs += fmt.Sprintln("\ncall by goroutine")
	}
	return &cs
}
