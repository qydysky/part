package component2

import (
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

func Get[TargetInterface any](id string, init ...func(TargetInterface) TargetInterface) (_interface TargetInterface) {
	if tmp, ok := pkgInterfaceMap[id].(TargetInterface); ok {
		for i := 0; i < len(init); i++ {
			tmp = init[i](tmp)
		}
		return tmp
	} else {
		panic(ErrGet.WithReason("ErrGet:" + id))
	}
}
