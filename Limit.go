package part

import (
	"time"
)

type Limitl struct {
	Stop bool
	Max int
	Second int
	TimeOut int
	Channl chan bool
}

func Limit(Max,Second,TimeOut int) (*Limitl) {

	returnVal := Limitl{}
	if Max < 1 || Second < 1 || TimeOut < Second{return &returnVal}

	returnVal = Limitl{
		Max:Max,
		Second:Second,
		TimeOut:TimeOut,
		Channl:make(chan bool,Max),
	}

	go func(returnVal *Limitl){
		for !returnVal.Stop {
			for i:=1;i<=Max;i++{
				returnVal.Channl <- true
			}
			time.Sleep(time.Duration(Second)*time.Second)
		}
	}(&returnVal)

	return &returnVal
}

func (l *Limitl) TO() bool {
	if l.Stop {return false}
	select {
		case <-l.Channl :;
		case <-time.After(time.Duration(l.TimeOut)*time.Second):return true;
	}
	return false
}