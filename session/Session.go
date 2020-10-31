package part

import (
	"time"
	"errors"
	"strconv"
)

type session struct {
	SumInTimeout int64
	Timeout int64
	session_now int64
}

var (
	session_ks map[string]string = make(map[string]string)
	session_kt map[string]int64 = make(map[string]int64)
	session_rand int64 = 1
	session_stop chan bool = make(chan bool,1)
)


func Session(SumInTimeout,Timeout int64) (*session,error) {
	if SumInTimeout == 0 {return &session{},errors.New("SumInTimeout == 0")}
	if Timeout == 0 {return &session{},errors.New("Timeout == 0")}

	s := new(session)

	if s.session_now == 0 {
		go func(){
			for{
				s.session_now = time.Now().Unix()
				time.Sleep(time.Second)
			}
		}()
	}

	return s,nil
}

func (s *session) Set(key string) (val string) {
	
	session_stop <- true

	if session_rand >= s.SumInTimeout {session_rand = 1}else{session_rand += 1}

	t := strconv.FormatInt(session_rand, 10)
	session_ks[t] = key
	session_kt[t] = s.session_now

	<-session_stop
	return t
}

func (s *session) Get(val string) (ok bool,key string){

	K, oks := session_ks[val]
	if !oks {return false,""}
	T, _ := session_kt[val]
	
	return s.session_now-T <= s.Timeout, K
}

func (s *session) Check(val string,key string) bool {
	ok,k := s.Get(val)
	return ok && k == key
}

func (s *session) Buf() (int64,int) {
	return s.session_now,len(session_ks)
}
