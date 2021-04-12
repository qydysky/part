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
	object := limit{
		maxNum_in_period:maxNum_in_period,
		ms_in_period:ms_in_period,
		ms_to_timeout:ms_to_timeout,
		channl:make(chan struct{},maxNum_in_period),
	}

	go func(object *limit){
		for object.maxNum_in_period > 0 {
			for i:=1;i<=object.maxNum_in_period;i++{
				object.channl <- struct{}{}
			}
			time.Sleep(time.Duration(object.ms_in_period)*time.Millisecond)
		}
	}(&object)

	return &object
}

// the func will return true if the request(call TO()) is up to limit and return false if not
func (l *limit) TO() bool {
	select {
		case <-l.channl :;
		case <-time.After(time.Duration(l.ms_to_timeout)*time.Millisecond):return true;
	}
	return false
}

//assert interface{} to *limit
func GetStruct(i interface{}) (l *limit,ok bool) {
	l,ok = i.(*limit)
	return 
}