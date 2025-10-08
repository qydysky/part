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

// 组件内部应自行处理协程安全
//
// 注册错误时，返回错误
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

// 组件内部应自行处理协程安全
//
// 注册错误时，抛出恐慌
func RegisterOrPanic[TargetInterface any](id string, _interface TargetInterface) {
	if e := Register(id, _interface); e != nil {
		panic(e)
	}
}

// Deprecated: use GetV3
//
// 接口可能未初始化
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

// Deprecated: use GetV3
//
// PreFunc 大多数不需要进行特别申明
func GetV2[TargetInterface any](id string, prefunc PreFunc[TargetInterface]) *GetRunner[TargetInterface] {
	runner := &GetRunner[TargetInterface]{cmpid: id}
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

// default prefunc PreFuncErr
func GetV3[TargetInterface any](id string, prefunc ...PreFunc[TargetInterface]) *GetRunner[TargetInterface] {
	if len(prefunc) == 0 {
		prefunc = append(prefunc, PreFuncErr[TargetInterface]{})
	}
	runner := &GetRunner[TargetInterface]{cmpid: id}
	if o, ok := pkgInterfaceMap[id]; !ok {
		runner.err = prefunc[0].ErrNoFound(id)
		return runner
	} else if tmp, ok := o.(TargetInterface); !ok {
		runner.err = prefunc[0].ErrTypeAssertion(id)
		return runner
	} else {
		runner.inter = prefunc[0].Init(tmp)
		return runner
	}
}

// Run:初始化失败时，返回err，不执行。内部返回错误时，返回err
//
// Run2:初始化失败时，不执行。内部错误需要通过赋值外部变量获取
//
// Run3:初始化失败时，传递error。内部错误需要通过赋值外部变量获取
type GetRunner[TargetInterface any] struct {
	cmpid string
	err   error
	inter TargetInterface
}

func (t *GetRunner[TargetInterface]) Err() error {
	return t.err
}

// 获取接口，当初始化失败时，调用ifFail进行初始化
//
// 当未传入ifFail时，将panic
func (t *GetRunner[TargetInterface]) Inter(ifFail ...func(error) TargetInterface) TargetInterface {
	if t.err != nil {
		if len(ifFail) == 0 {
			p := PreFuncPanic[TargetInterface]{}
			if ErrNoFound.Catch(t.err) {
				_ = p.ErrNoFound(t.cmpid)
			}
			if ErrTypeAssertion.Catch(t.err) {
				_ = p.ErrTypeAssertion(t.cmpid)
			}
			panic(t.cmpid)
		} else {
			return ifFail[0](t.err)
		}
	}
	return t.inter
}

// 初始化失败时，返回err，不执行。内部返回错误时，返回err
func (t *GetRunner[TargetInterface]) Run(f func(TargetInterface) error) error {
	if t.err != nil {
		return t.err
	}
	return f(t.inter)
}

// Run2:初始化失败时，不执行。内部错误需要通过赋值外部变量获取
func (t *GetRunner[TargetInterface]) Run2(f func(inter TargetInterface)) {
	if t.err == nil {
		f(t.inter)
	}
}

// Run3:初始化失败时，传递error。内部错误需要通过赋值外部变量获取
func (t *GetRunner[TargetInterface]) Run3(f func(inter TargetInterface, e error)) {
	f(t.inter, t.err)
}

// PreFuncCu[TargetInterface any] 自定义的初始化，错误处理
//
// PreFuncErr[TargetInterface any] 无初始化，错误处理为调用时返回error
//
// PreFuncPanic[TargetInterface any] 无初始化，错误处理为立刻panic
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
