package part

import (
	"sync/atomic"
	"time"

	signal "github.com/qydysky/part/signal"
)

type Limit struct {
	druation             time.Duration
	druation_timeout     time.Duration
	bucket               chan struct{}
	wait_num             atomic.Int32
	pre_bucket_token_num atomic.Int64
	maxNum_in_period     int
	cancel               *signal.Signal
}

// create a Limit Object
// it will allow maxNum_in_period requests(call TO()) in ms_in_period. if the request(call TO()) is out of maxNum_in_period,it will wait ms_to_timeout
// ms_to_timeout [>0:will wait ms] [=0:no wait] [<0:will block]
func New(maxNum_in_period int, druation, druation_timeout string) *Limit {
	object := Limit{
		bucket: make(chan struct{}, maxNum_in_period),
		cancel: signal.Init(),
	}

	if maxNum_in_period <= 0 {
		panic("maxNum_in_period <= 0")
	} else {
		object.maxNum_in_period = maxNum_in_period
	}

	if t, e := time.ParseDuration(druation); e != nil {
		panic(e)
	} else {
		object.druation = t.Abs()
	}

	if t, e := time.ParseDuration(druation_timeout); e != nil {
		panic(e)
	} else {
		object.druation_timeout = t
	}

	for i := 0; i < maxNum_in_period; i++ {
		object.bucket <- struct{}{}
	}

	go func() {
		ec, fin := object.cancel.WaitC()
		defer fin()
		defer close(object.bucket)

		if object.druation == 0 {
			for {
				select {
				case object.bucket <- struct{}{}:
				case <-ec:
					return
				}
			}
		} else {
			for {
				select {
				case <-time.After(object.druation):
				case <-ec:
					return
				}

				object.pre_bucket_token_num.Store(int64(len(object.bucket)))

				for i := object.maxNum_in_period; i > 0; i-- {
					select {
					case object.bucket <- struct{}{}:
					case <-ec:
						return
					default:
						i = 0
					}
				}
			}
		}
	}()

	return &object
}

// the func will return true if the request(call TO()) is up to limit and return false if not
func (l *Limit) TO() bool {
	if !l.IsLive() {
		return false
	}

	l.wait_num.Add(1)
	defer l.wait_num.Add(-1)

	if l.druation_timeout < 0 || l.druation == 0 {
		<-l.bucket
		return false
	} else if l.druation_timeout == 0 {
		select {
		case <-l.bucket:
			return false
		default:
			return true
		}
	} else {
		select {
		case <-l.bucket:
			return false
		case <-time.NewTimer(l.druation_timeout).C:
			return true
		}
	}
}

func (l *Limit) IsLive() bool {
	return l.cancel.Islive()
}

func (l *Limit) Close() {
	l.cancel.Done()
}

// return the token number of bucket at now
func (l *Limit) TK() int {
	return len(l.bucket)
}

// return the token number of bucket at previous
func (l *Limit) PTK() int {
	return int(l.pre_bucket_token_num.Load())
}

func (l *Limit) WNum() int32 {
	return l.wait_num.Load()
}
