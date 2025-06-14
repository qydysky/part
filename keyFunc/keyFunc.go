package part

import (
	"errors"
	"iter"

	perrors "github.com/qydysky/part/errors"
)

var (
	ErrKeyMissAgain = errors.New(`ErrKeyMissAgain`)
	ErrNextMethod   = perrors.Action(`ErrNextMethod`)
	ErrKeyNotReg    = errors.New(`ErrKeyNotReg`)
)

type KeyFunc struct {
	keyGet   map[string][]func() (misskey string, err error)
	keyCheck map[string]func() bool
}

func NewKeyFunc() *KeyFunc {
	return &KeyFunc{
		keyGet:   make(map[string][]func() (misskey string, err error)),
		keyCheck: make(map[string]func() bool),
	}
}

// checkf func() bool
//
//	当key有效时，应返回true,否则false
//
// fs func() (misskeys string, err error):
//
//	若缺失依赖key,应返回misskey!=""
//	若方法错误，但不是严重错误，应返回misskey=="" err==ErrNextMethod.NewErr(err)，以便尝试下一个fs
//	若方法错误，需要立即退出，应返回misskey=="" err!=nil
//	其他情况，应返回misskey=="" err==nil
func (t *KeyFunc) Reg(key string, checkf func() bool, fs ...func() (misskey string, err error)) *KeyFunc {
	t.keyGet[key] = fs
	t.keyCheck[key] = checkf
	return t
}

func (t *KeyFunc) Get(key string) (err error) {
	return t.GetTrace(key).Err
}

type Node struct {
	Key         string
	MethodIndex int
	Err         error
	perp        any
	nextp       any
}

func newNode(key string, methodI int, err error) *Node {
	tmp := &Node{
		Key:         key,
		MethodIndex: methodI,
		Err:         err,
	}
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

func (t *Node) Asc() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for i := t.First(); i != nil; i = i.Next() {
			if !yield(i) {
				return
			}
		}
	}
}

func (t *Node) Desc() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for i := t.Last(); i != nil; i = i.Per() {
			if !yield(i) {
				return
			}
		}
	}
}

func (t *KeyFunc) GetTrace(key string) *Node {
	return t.getTrace(key, nil)
}

func (t *KeyFunc) getTrace(key string, trace *Node) *Node {
	if fs, ok := t.keyGet[key]; !ok || len(fs) == 0 {
		return newNode(key, 0, ErrKeyNotReg)
	} else {
		for i := 0; i < len(fs); i++ {
			if trace == nil {
				trace = newNode(key, i, nil)
			} else {
				tmpp := newNode(key, i, nil)
				trace.nextp = tmpp
				tmpp.perp = trace
				trace = tmpp
			}
			if t.keyCheck[key]() {
				return trace
			}
			missKey, err := fs[i]()
			trace.Err = err
			if missKey != "" {
				for node := range trace.Desc() {
					if node.Key == missKey {
						trace.Err = ErrKeyMissAgain
						return trace
					}
				}
				trace = t.getTrace(missKey, trace)
			}
			if trace.Err == nil {
				if missKey != "" {
					i -= 1
					continue
				}
				return trace
			} else if perrors.Catch(trace.Err, ErrNextMethod) {
				continue
			} else {
				return trace
			}
		}
		return trace
	}
}
