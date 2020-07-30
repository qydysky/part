package part

import (
	"time"
	"strconv"
)

type session struct {}

const (
	session_SumInTimeout int64 = 1e5
	session_Timeout int64 = 1
)

var (
	session_ks map[string]string = make(map[string]string)
	session_kt map[string]int64 = make(map[string]int64)
	session_rand int64 = 1
	session_now int64
	session_stop chan bool = make(chan bool,1)
)


func Session() (*session) {return &session{}}

func (s *session) Set(key string) (val string) {
	
	session_stop <- true

	if session_rand >= session_SumInTimeout {session_rand = 1}else{session_rand += 1}

	t := strconv.FormatInt(session_rand, 10)
	session_ks[t] = key
	session_kt[t] = session_now

	<-session_stop
	return t
}

func (s *session) Get(val string) (ok bool,key string){

	K, oks := session_ks[val]
	if !oks {return false,""}
	T, _ := session_kt[val]
	
	return session_now-T <= session_Timeout, K
}

func (s *session) Check(val string,key string) bool {
	ok,k := s.Get(val)
	return ok && k == key
}

func (s *session) Buf() (int64,int) {
	return session_now,len(session_ks)
}


func init(){
	go func(){
		for{
			session_now = time.Now().Unix()
			time.Sleep(time.Second)
		}
	}()
}