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

	// Logf().NoShow(false)

	if Max < 1 {
		Logf().E("Limit:Max < 1 is true.Set to 1")
		Max = 1
	}

	returnVal := Limitl{
		Max:Max,
		Channl:make(chan bool,Max),
	}

	if Second < 1 || TimeOut < Second{
		Logf().E("Limit:Second < 1 || TimeOut < Second is true.Set Stop to true")
		returnVal.Stop = true
		return &returnVal
	}

	returnVal = Limitl{
		Second:Second,
		TimeOut:TimeOut,
	}

	go func(returnVal *Limitl){
		for !returnVal.Stop {
			for i:=1;i<=Max;i++{
				returnVal.Channl <- true
			}
			time.Sleep(time.Duration(Second)*time.Millisecond)
		}
	}(&returnVal)

	return &returnVal
}

func (l *Limitl) TO() bool {
	if l.Stop {return false}
	select {
		case <-l.Channl :;
		case <-time.After(time.Duration(l.TimeOut)*time.Millisecond):return true;
	}
	return false
}