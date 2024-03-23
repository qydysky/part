package bools

import (
	"errors"
)

type sr struct {
	rule string
	i    int
	fin  bool
	arg  map[string]func() bool
}

const (
	emp = iota
	and
	or
	not
)

var (
	ErrNoAct      = errors.New("ErrNoAct")
	ErrHadAct     = errors.New("ErrHadAct")
	ErrUnkownChar = errors.New("ErrUnkownChar")
)

func New(s string, arg map[string]func() bool) *sr {
	return &sr{s, 0, false, arg}
}

func (t *sr) SetRule(s string) *sr {
	t.rule = s
	t.i = 0
	t.fin = t.i == len(t.rule)
	return t
}

func (t *sr) SetArg(arg map[string]func() bool) *sr {
	t.arg = arg
	t.i = 0
	t.fin = t.i == len(t.rule)
	return t
}

func (t *sr) next() (b byte) {
	b = t.rule[t.i]
	t.i = t.i + 1
	t.fin = t.i == len(t.rule)
	return
}

func (t *sr) Check() (result bool, err error) {
	result = true
	act := and
	no := emp
	for !t.fin {
		switch t.next() {
		case '{':
			switch act {
			case and:
				if result {
					if no == not {
						result = result && !t.parseArg()
					} else {
						result = result && t.parseArg()
					}
				} else {
					t.skipTo('}')
					return false, nil
				}
				no = emp
				act = emp
			case or:
				if result {
					t.skipTo('}')
					return true, nil
				} else {
					if no == not {
						result = result || !t.parseArg()
					} else {
						result = result || t.parseArg()
					}
				}
				no = emp
				act = emp
			default:
				return false, ErrNoAct
			}
		case '!':
			if no == emp {
				no = not
			} else {
				no = emp
			}
		case '&':
			if act != emp || no != emp {
				return false, ErrHadAct
			}
			act = and
		case '|':
			if act != emp || no != emp {
				return false, ErrHadAct
			}
			act = or
		case '(':
			switch act {
			case and:
				if result {
					if cr, e := t.Check(); e != nil {
						return false, e
					} else {
						if no == not {
							result = result && !cr
						} else {
							result = result && cr
						}
					}
				} else {
					t.skipTo(')')
					return false, nil
				}
				no = emp
				act = emp
			case or:
				if result {
					t.skipTo(')')
					return true, nil
				} else {
					if cr, e := t.Check(); e != nil {
						return false, e
					} else {
						if no == not {
							result = result || !cr
						} else {
							result = result || cr
						}
					}
				}
				no = emp
				act = emp
			default:
				return false, ErrNoAct
			}
		case ')':
			return
		case '\t':
		case '\n':
		case ' ':
		default:
			return false, ErrUnkownChar
		}
	}
	return
}

func (t *sr) parseArg() bool {
	cu := t.i
	for !t.fin {
		switch t.next() {
		case '}':
			argB, ok := t.arg[t.rule[cu:t.i-1]]
			return argB() && ok
		default:
		}
	}
	return false
}

func (t *sr) skipTo(b byte) {
	for !t.fin && t.next() != b {
	}
}

func True() bool {
	return true
}

func False() bool {
	return false
}
