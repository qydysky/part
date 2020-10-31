package part

import (
	"time"
	"errors"
	"strconv"
)

type session struct {
	SumInSecond int64
	Timeout int64
	session_now int64
	session_rand int64
	session_ks map[string]string
	session_kt map[string]int64
	session_stop chan bool
}

func Session(SumInSecond,Timeout int64) (*session,error) {
	if SumInSecond == 0 {return &session{},errors.New("SumInTimeout == 0")}
	if Timeout == 0 {return &session{},errors.New("Timeout == 0")}

	s := new(session)

	if s.session_now == 0 {
		s.session_rand = 1
		s.session_ks = make(map[string]string)
		s.session_kt = make(map[string]int64)
		s.session_stop = make(chan bool,1)
		s.SumInSecond = SumInSecond
		s.Timeout = Timeout
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
	
	s.session_stop <- true

	if s.session_rand >= s.SumInTimeout {s.session_rand = 1}else{s.session_rand += 1}

	t := strconv.FormatInt(s.session_rand, 10)
	s.session_ks[t] = key
	s.session_kt[t] = s.session_now

	<-s.session_stop
	return t
}

func (s *session) Get(val string) (ok bool,key string){

	K, oks := s.session_ks[val]
	if !oks {return false,""}
	T, _ := s.session_kt[val]
	
	return s.session_now-T <= s.Timeout, K
}

func (s *session) Check(val string,key string) bool {
	ok,k := s.Get(val)
	return ok && k == key
}

func (s *session) Buf() (int64,int) {
	return s.session_now,len(s.session_ks)
}
