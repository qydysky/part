package component2

import (
	"os"
	"runtime"
	"strings"

	perrors "github.com/qydysky/part/errors"
)

var pkgInterfaceMap = make(map[string]any)

var (
	ErrEmptyPkgId = perrors.New("ErrEmptyPkgId")
	ErrRegistered = perrors.New("ErrRegistered")
	ErrGet        = perrors.New("ErrGet")
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

type PreFunc[TargetInterface any] interface {
	Init(TargetInterface) TargetInterface
	ErrNoFound(id string)
	ErrTypeAssertion(id string)
}

type PreFuncPanic[TargetInterface any] struct{}

func (PreFuncPanic[TargetInterface]) Init(s TargetInterface) TargetInterface {
	return s
}

func (PreFuncPanic[TargetInterface]) ErrNoFound(id string) {
	panic(ErrGet.WithReason("ErrNoFound"))
}

func (PreFuncPanic[TargetInterface]) ErrTypeAssertion(id string) {
	panic(ErrGet.WithReason("ErrTypeAssertion"))
}

type PreFuncErr[TargetInterface any] struct{}

func (PreFuncErr[TargetInterface]) Init(s TargetInterface) TargetInterface {
	return s
}

func (PreFuncErr[TargetInterface]) ErrNoFound(id string) {
	os.Stderr.WriteString(perrors.ErrorFormat(ErrGet.WithReason("ErrNoFound "+id), perrors.ErrActionSimplifyFunc))
}

func (PreFuncErr[TargetInterface]) ErrTypeAssertion(id string) {
	os.Stderr.WriteString(perrors.ErrorFormat(ErrGet.WithReason("ErrTypeAssertion "+id), perrors.ErrActionSimplifyFunc))
}

type PreFuncCu[TargetInterface any] struct {
	Initf             func(TargetInterface) TargetInterface
	ErrNoFoundf       func(id string)
	ErrTypeAssertionf func(id string)
}

func (t PreFuncCu[TargetInterface]) Init(s TargetInterface) TargetInterface {
	if t.Initf != nil {
		return t.Initf(s)
	}
	return s
}

func (t PreFuncCu[TargetInterface]) ErrNoFound(id string) {
	if t.ErrNoFoundf != nil {
		t.ErrNoFoundf(id)
	}
}

func (t PreFuncCu[TargetInterface]) ErrTypeAssertion(id string) {
	if t.ErrTypeAssertionf != nil {
		t.ErrTypeAssertionf(id)
	}
}
