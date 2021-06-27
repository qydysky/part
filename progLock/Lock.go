package part

import (
	"os"
	"sync"
	"time"
	"errors"
	"fmt"
	"encoding/json"

	part "github.com/qydysky/part"
)

type lock struct {
	Time int64 `json:"t"`
	stopsign bool
	b chan struct{}
	sync.Mutex
}

const (
	lock_file = ".lock"
	lock_timeout = 10
)

func New() *lock {
	return &lock{}
}

func (l *lock) Start() (err error) {
	l.Lock()
	defer l.Unlock()

	if part.Checkfile().IsExist(lock_file) {
		//read time from file
		if s := part.File().FileWR(part.Filel{
			File:lock_file,
			Loc:0,
		});len(s) != 0 {
			if e := json.Unmarshal([]byte(s), l);e != nil {
				err = e
			}
		} else {err = errors.New("read error")}
		//read time from modtime
		if err != nil {
			err,l.Time = part.Checkfile().GetFileModTime(lock_file)
		}
		
		if err != nil {panic(err.Error())}

		if time.Now().Unix() - l.Time <= lock_timeout {
			return errors.New("still alive")
		}
	} else {
		l.b = make(chan struct{})
		go func(l *lock){
			for !l.stopsign {
				l.Time = time.Now().Unix()
				if b,e := json.Marshal(l);e != nil {
					panic(e.Error())
				} else {
					part.File().FileWR(part.Filel{
						File:lock_file,
						Loc:0,
						Context:[]interface{}{b},
					})
				}

				select{
				case l.b<-struct{}{}:;
				default:;
				}

				time.Sleep(time.Duration(lock_timeout)*time.Second)
			}
		}(l)
		<- l.b
	}
	return nil
}

func (l *lock) Stop() error {
	l.Lock()
	defer l.Unlock()
	
	l.stopsign = true
	close(l.b)
	return os.RemoveAll(lock_file)
}