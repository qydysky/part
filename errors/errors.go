package errors

type Error struct {
	son    interface{}
	reason string
	action string
}

func (t Error) Error() string {
	return t.reason
}

func Catch(e error, action string) bool {
	if v, ok := e.(Error); ok {
		if v.action == action {
			return true
		} else if v.son != nil {
			return Catch((v.son).(Error), action)
		}
	}
	return false
}

func Grow(e error, son Error) Error {
	if v, ok := e.(Error); ok {
		son.son = v
	} else {
		son.son = Error{
			reason: v.Error(),
		}
	}
	return son
}

func New(reason string, action string) Error {
	return Error{
		reason: reason,
		action: action,
	}
}
