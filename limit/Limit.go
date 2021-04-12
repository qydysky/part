package part

import (
	"time"
)

type Limit struct {
	maxNum_in_period int
	ms_in_period int
	ms_to_timeout int
	bucket chan struct{}
	pre_bucket_token_num int
}

// create a Limit Object
// it will allow maxNum_in_period requests(call TO()) in ms_in_period. if the request(call TO()) is out of maxNum_in_period,it will wait ms_to_timeout
func New(maxNum_in_period,ms_in_period,ms_to_timeout int) (*Limit) {
	object := Limit{
		maxNum_in_period:maxNum_in_period,
		ms_in_period:ms_in_period,
		ms_to_timeout:ms_to_timeout,
		bucket:make(chan struct{},maxNum_in_period),
	}

	go func(object *Limit){
		for object.maxNum_in_period > 0 {
			object.bucket <- struct{}{}
			for i:=1;i<object.maxNum_in_period;i++{
				select {
				case object.bucket <- struct{}{}:;
				default :i = object.maxNum_in_period
				}
			}
			time.Sleep(time.Duration(object.ms_in_period)*time.Millisecond)
			object.pre_bucket_token_num = len(object.bucket)
		}
	}(&object)

	//make sure the bucket is full
	for object.TK() != maxNum_in_period {}
	object.pre_bucket_token_num = len(object.bucket)
	
	return &object
}

// the func will return true if the request(call TO()) is up to limit and return false if not
func (l *Limit) TO() bool {
	select {
		case <-l.bucket :;
		case <-time.After(time.Duration(l.ms_to_timeout)*time.Millisecond):return true;
	}
	return false
}

// return the token number of bucket at now
func (l *Limit) TK() int {
	return len(l.bucket)
}

// return the token number of bucket at previous
func (l *Limit) PTK() int {
	return l.pre_bucket_token_num
}