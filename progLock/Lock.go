package part

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	part "github.com/qydysky/part"
	pfile "github.com/qydysky/part/file"
)

type lock struct {
	Time     int64  `json:"time"`
	Data     string `json:"date"`
	stopsign bool
	b        chan struct{}
	sync.Mutex
}

const (
	lock_file    = ".lock"
	lock_timeout = 10
)

func New() *lock {
	return &lock{}
}

// start will save current time and Data
// if start func is called in other programe, Data or "still alive" will return as error
func (l *lock) Start(Data ...string) (err error) {
	l.Lock()
	defer l.Unlock()

	f := pfile.New(lock_file, 0, true)
	if f.IsExist() {
		//read time from file
		if s := part.File().FileWR(part.Filel{
			File: lock_file,
			Loc:  0,
		}); len(s) != 0 {
			if e := json.Unmarshal([]byte(s), l); e != nil {
				err = e
			}
		} else {
			err = errors.New("read error")
		}
		//read time from modtime
		if err != nil {
			l.Time, err = f.GetFileModTime()
		}

		if err != nil {
			panic(err.Error())
		}

		if time.Now().Unix()-l.Time <= lock_timeout {
			if l.Data != "" {
				return errors.New(l.Data)
			}
			return errors.New("still alive")
		}
	} else {
		l.b = make(chan struct{})
		l.Data = strings.Join(Data, "&")
		go func(l *lock) {
			for !l.stopsign {
				l.Time = time.Now().Unix()
				if b, e := json.Marshal(l); e != nil {
					panic(e.Error())
				} else {
					part.File().FileWR(part.Filel{
						File:    lock_file,
						Loc:     0,
						Context: []interface{}{b},
					})
				}

				select {
				case l.b <- struct{}{}:
				default:
				}

				time.Sleep(time.Duration(lock_timeout) * time.Second)
			}
		}(l)
		<-l.b
	}
	return nil
}

func (l *lock) Stop() error {
	l.Lock()
	defer l.Unlock()

	l.stopsign = true
	if l.b != nil {
		close(l.b)
		l.b = nil
	}
	return os.RemoveAll(lock_file)
}
