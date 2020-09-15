package part

import (
	"time"
)

type Limitl struct {
	Stop bool
	Max int
	Millisecond int
	MilliTimeOut int
	Channl chan bool
}

func Limit(Max,Millisecond,MilliTimeOut int) (*Limitl) {

	// Logf().NoShow(false)

	if Max < 1 {
		Logf().E("Limit:Max < 1 is true.Set to 1")
		Max = 1
	}

	returnVal := Limitl{
		Max:Max,
		Channl:make(chan bool,Max),
	}

	if Millisecond < 1 || MilliTimeOut < Millisecond{
		Logf().E("Limit:Millisecond < 1 || MilliTimeOut < Millisecond is true.Set Stop to true")
		returnVal.Stop = true
		return &returnVal
	}

	returnVal.Millisecond=Millisecond
	returnVal.MilliTimeOut=MilliTimeOut

	go func(returnVal *Limitl){
		for !returnVal.Stop {
			for i:=1;i<=returnVal.Max;i++{
				returnVal.Channl <- true
			}
			time.Sleep(time.Duration(Millisecond)*time.Millisecond)
		}
	}(&returnVal)

	return &returnVal
}

func (l *Limitl) TO() bool {
	if l.Stop {return false}
	select {
		case <-l.Channl :;
		case <-time.After(time.Duration(l.MilliTimeOut)*time.Millisecond):return true;
	}
	return false
}