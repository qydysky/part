package part

import (
	"errors"
	"iter"
)

var (
	ErrGetMissKeyFail = errors.New(`ErrGetMissKeyFail`)
	ErrNextMethod     = errors.New(`ErrNextMethod`)
	ErrKeyNotReg      = errors.New(`ErrKeyNotReg`)
)

type Api struct {
	keyGet map[string][]func() (misskey string, err error)
}

func NewApi() *Api {
	return &Api{
		keyGet: make(map[string][]func() (misskey string, err error)),
	}
}

// fs func() (misskeys string, err error):
//
//	当 misskeys == "" 时，
//	若 err == nil 表示key有效，Get(key) 将会退出
//	若 err == ErrNextMethod 则会尝试下一个fs，没有下个fs时 err = ErrAllMethodFail，其他 err Get(key) 将会退出并返回
//
// 当 misskeys != "" 时， Get(key) 将尝试 Get(misskey)
func (t *Api) Reg(key string, fs ...func() (misskey string, err error)) *Api {
	t.keyGet[key] = fs
	return t
}

func (t *Api) Get(key string) (err error) {
	if fs, ok := t.keyGet[key]; !ok || len(fs) == 0 {
		return ErrKeyNotReg
	} else {
		var lastMisskey string
		for i := 0; i < len(fs); i++ {
			missKey, err := fs[i]()
			if missKey != "" {
				if lastMisskey == missKey {
					return ErrGetMissKeyFail
				}
				lastMisskey = missKey
				err = t.Get(missKey)
			}
			if err == nil {
				if missKey != "" {
					i -= 1
					continue
				}
				return nil
			} else if errors.Is(err, ErrNextMethod) {
				continue
			} else {
				return err
			}
		}
		return
	}
}

type Node struct {
	Key         string
	MethodIndex int
	Err         error
	perp        any
	cup         any
	nextp       any
}

func NewNode(key string, methodI int, err error) *Node {
	tmp := &Node{
		Key:         key,
		MethodIndex: methodI,
		Err:         err,
	}
	tmp.cup = tmp
	return tmp
}

func (t *Node) First() (f *Node) {
	f = t
	for f.perp != nil {
		f = f.perp.(*Node)
	}
	return
}

func (t *Node) Last() (f *Node) {
	f = t
	for f.nextp != nil {
		f = f.nextp.(*Node)
	}
	return
}

func (t *Node) Per() *Node {
	if p, ok := t.perp.(*Node); ok {
		return p
	}
	return nil
}

func (t *Node) Next() *Node {
	if p, ok := t.nextp.(*Node); ok {
		return p
	}
	return nil
}

func (t *Node) CallTree() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for i := t.First(); i != nil; i = i.Next() {
			if !yield(i) {
				return
			}
		}
	}
}

func (t *Api) GetTrace(key string) *Node {
	if fs, ok := t.keyGet[key]; !ok || len(fs) == 0 {
		return NewNode(key, 0, ErrKeyNotReg)
	} else {
		var lastMisskey string
		var trace *Node
		for i := 0; i < len(fs); i++ {
			if trace == nil {
				trace = NewNode(key, i, nil)
			} else {
				tmpp := NewNode(key, i, nil)
				trace.nextp = tmpp
				tmpp.perp = trace
				trace = tmpp
			}
			missKey, err := fs[i]()
			trace.Err = err
			if missKey != "" {
				if lastMisskey == missKey {
					trace.Err = ErrGetMissKeyFail
					return trace
				}
				lastMisskey = missKey

				tmpp := t.GetTrace(missKey)
				trace.nextp = tmpp.First()
				tmpp.First().perp = trace
				trace = tmpp
			}
			if trace.Err == nil {
				if missKey != "" {
					i -= 1
					continue
				}
				return trace
			} else if errors.Is(trace.Err, ErrNextMethod) {
				continue
			} else {
				return trace
			}
		}
		return trace
	}
}
