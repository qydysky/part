package component2

import (
	"errors"
	"runtime"
	"strings"
)

var pkgInterfaceMap = make(map[string]any)

var (
	ErrEmptyPkgId = errors.New("ErrEmptyPkgId")
	ErrRegistered = errors.New("ErrRegistered")
	ErrGet        = errors.New("ErrGet")
)

func PkgId() string {
	if pc, _, _, ok := runtime.Caller(1); ok {
		return strings.TrimSuffix(runtime.FuncForPC(pc).Name(), ".init")
	}
	return ""
}

func Register[TargetInterface any](pkgId string, _interface TargetInterface) error {
	if pkgId == "" {
		return ErrEmptyPkgId
	}
	if _interfaceReg, ok := pkgInterfaceMap[pkgId]; ok && _interfaceReg != nil {
		return ErrRegistered
	} else {
		pkgInterfaceMap[pkgId] = _interface
	}
	return nil
}

func Get[TargetInterface any](pkgId string, init ...func(TargetInterface) TargetInterface) (_interface TargetInterface) {
	if tmp, ok := pkgInterfaceMap[pkgId].(TargetInterface); ok {
		for i := 0; i < len(init); i++ {
			tmp = init[i](tmp)
		}
		return tmp
	} else {
		panic(ErrGet)
	}
}
