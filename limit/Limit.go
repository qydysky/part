package part

import (
	"time"
)

type limit struct {
	maxNum_in_period int
	ms_in_period int
	ms_to_timeout int
	channl chan struct{}
}

// create a Limit Object
// it will allow maxNum_in_period requests(call TO()) in ms_in_period. if the request(call TO()) is out of maxNum_in_period,it will wait ms_to_timeout
func New(maxNum_in_period,ms_in_period,ms_to_timeout int) (*limit) {
	if maxNum_in_period < 1 {panic(`limit max < 1`)}

	returnVal := limit{
		maxNum_in_period:maxNum_in_period,
		ms_in_period:ms_in_period,
		ms_to_timeout:ms_to_timeout,
		channl:make(chan struct{},maxNum_in_period),
	}

	go func(returnVal *limit){
		for {
			for i:=1;i<=returnVal.maxNum_in_period;i++{
				returnVal.channl <- struct{}{}
			}
			time.Sleep(time.Duration(ms_in_period)*time.Millisecond)
		}
	}(&returnVal)

	return &returnVal
}

// the func will return true if the request(call TO()) is up to limit and return false if not
func (l *limit) TO() bool {
	select {
		case <-l.channl :;
		case <-time.After(time.Duration(l.ms_to_timeout)*time.Millisecond):return true;
	}
	return false
}