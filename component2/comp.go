package component2

import (
	"runtime"
	"strings"

	perrors "github.com/qydysky/part/errors"
)

var pkgInterfaceMap = make(map[string]any)

var (
	ErrEmptyPkgId    = perrors.New("ErrEmptyPkgId")
	ErrRegistered    = perrors.New("ErrRegistered")
	ErrNoFound       = perrors.Action("ErrNoFound")
	ErrTypeAssertion = perrors.Action("ErrTypeAssertion")
)

func PkgId(varId ...string) string {
	if pc, _, _, ok := runtime.Caller(1); ok {
		return strings.Join(append([]string{strings.TrimSuffix(runtime.FuncForPC(pc).Name(), ".init")}, varId...), ".")
	}
	return ""
}

func Register[TargetInterface any](id string, _interface TargetInterface) error {
	if id == "" {
		return ErrEmptyPkgId
	}
	if _interfaceReg, ok := pkgInterfaceMap[id]; ok && _interfaceReg != nil {
		return ErrRegistered
	} else {
		pkgInterfaceMap[id] = _interface
	}
	return nil
}

func RegisterOrPanic[TargetInterface any](id string, _interface TargetInterface) {
	if e := Register(id, _interface); e != nil {
		panic(e)
	}
}

func Get[TargetInterface any](id string, prefunc ...PreFunc[TargetInterface]) (_interface TargetInterface) {
	if len(prefunc) == 0 {
		prefunc = append(prefunc, PreFuncErr[TargetInterface]{})
	}
	if o, ok := pkgInterfaceMap[id]; !ok {
		for i := 0; i < len(prefunc); i++ {
			prefunc[i].ErrNoFound(id)
		}
	} else if tmp, ok := o.(TargetInterface); ok {
		for i := 0; i < len(prefunc); i++ {
			tmp = prefunc[i].Init(tmp)
		}
		return tmp
	} else {
		for i := 0; i < len(prefunc); i++ {
			prefunc[i].ErrTypeAssertion(id)
		}
	}
	return *new(TargetInterface)
}

func GetV2[TargetInterface any](id string, prefunc PreFunc[TargetInterface]) *GetRunner[TargetInterface] {
	runner := &GetRunner[TargetInterface]{}
	if o, ok := pkgInterfaceMap[id]; !ok {
		runner.err = prefunc.ErrNoFound(id)
		return runner
	} else if tmp, ok := o.(TargetInterface); !ok {
		runner.err = prefunc.ErrTypeAssertion(id)
		return runner
	} else {
		runner.inter = prefunc.Init(tmp)
		return runner
	}
}

type GetRunner[TargetInterface any] struct {
	err   error
	inter TargetInterface
}

func (t *GetRunner[TargetInterface]) Err() error {
	return t.err
}

func (t *GetRunner[TargetInterface]) Run(f func(TargetInterface) error) error {
	if t.err != nil {
		return t.err
	}
	return f(t.inter)
}

// PreFuncCu[TargetInterface any]
//
// PreFuncErr[TargetInterface any]
//
// PreFuncPanic[TargetInterface any]
type PreFunc[TargetInterface any] interface {
	Init(TargetInterface) TargetInterface
	ErrNoFound(id string) error
	ErrTypeAssertion(id string) error
}

type PreFuncPanic[TargetInterface any] struct{}

func (PreFuncPanic[TargetInterface]) Init(s TargetInterface) TargetInterface {
	return s
}

func (PreFuncPanic[TargetInterface]) ErrNoFound(id string) error {
	panic(perrors.ErrorFormat(ErrNoFound.New(id), perrors.ErrActionInLineFunc))
}

func (PreFuncPanic[TargetInterface]) ErrTypeAssertion(id string) error {
	panic(perrors.ErrorFormat(ErrTypeAssertion.New(id), perrors.ErrActionInLineFunc))
}

type PreFuncErr[TargetInterface any] struct{}

func (PreFuncErr[TargetInterface]) Init(s TargetInterface) TargetInterface {
	return s
}

func (PreFuncErr[TargetInterface]) ErrNoFound(id string) error {
	return ErrNoFound.New(id)
}

func (PreFuncErr[TargetInterface]) ErrTypeAssertion(id string) error {
	return ErrTypeAssertion.New(id)
}

type PreFuncCu[TargetInterface any] struct {
	Initf             func(TargetInterface) TargetInterface
	ErrNoFoundf       func(id string) error
	ErrTypeAssertionf func(id string) error
}

func (t PreFuncCu[TargetInterface]) Init(s TargetInterface) TargetInterface {
	if t.Initf != nil {
		return t.Initf(s)
	}
	return s
}

func (t PreFuncCu[TargetInterface]) ErrNoFound(id string) error {
	if t.ErrNoFoundf != nil {
		return t.ErrNoFoundf(id)
	} else {
		return ErrNoFound.New(id)
	}
}

func (t PreFuncCu[TargetInterface]) ErrTypeAssertion(id string) error {
	if t.ErrTypeAssertionf != nil {
		return t.ErrTypeAssertionf(id)
	} else {
		return ErrTypeAssertion.New(id)
	}
}
