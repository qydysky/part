package part

import (
	"os"
	"sync"
	"time"
	"errors"
)

type lock struct {
	sync.Mutex
}

var (
	lock_file string
	lock_timeout int64 = 10
)

func Lock() *lock {
	return &lock{}
}

func (l *lock) Start(filePath string,timeout int64) error {
	l.Lock()
	defer l.Unlock()

	if Checkfile().IsExist(filePath) {
		if e,t := Checkfile().GetFileModTime(filePath); e != nil || time.Now().Unix() - t <= lock_timeout {
			Logf().E(e.Error(),"or still alive")
			return errors.New("still alive")
		}
	}


	lock_file = filePath
	lock_timeout = timeout

	go func(){
		for lock_file != "" {
			File().FileWR(Filel{
				File:filePath,
				Loc:0,
				Context:[]interface{}{"still alive"},
			})
			Sys().Timeoutf(int(lock_timeout))
		}
	}()

	return nil
}

func (l *lock) Stop() {
	l.Lock()
	defer l.Unlock()

	os.RemoveAll(lock_file)

	lock_file = ""
	
}