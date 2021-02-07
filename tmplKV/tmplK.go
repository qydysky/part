package part

import (
	"time"
	"sync"
	idpool "github.com/qydysky/part/idpool"
)

type tmplK struct {
	SumInDruation int64
	Druation int64
	now int64
	pool *idpool.Idpool
	kvt_map map[string]tmplK_item
	sync.RWMutex
}

type tmplK_item struct {
	kv uintptr
	kt int64
	uid *idpool.Id
}

func New_tmplK(SumInDruation,Druation int64) (*tmplK) {

	s := &tmplK{
		SumInDruation:SumInDruation,
		Druation:Druation,
		kvt_map:make(map[string]tmplK_item),
		pool:idpool.New(),
	}
	go func(){
		ticker := time.NewTicker(time.Second)
		for{
			s.now = (<- ticker.C).Unix()
			go func(){
				tmp := make(map[string]tmplK_item)
				s.Lock()
				for k,v := range s.kvt_map {tmp[k] = v}
				s.kvt_map = tmp
				s.Unlock()
			}()
		}
	}()

	return s
}

func (s *tmplK) Set(key string) (id uintptr) {
	s.Lock()
	defer s.Unlock()
	
	if tmp, oks := s.kvt_map[key];oks {
		defer s.pool.Put(tmp.uid)//在取得新Id后才put回
	} else if s.SumInDruation >= 0 && s.pool.Len() >= uint(s.SumInDruation){//不为无限&&达到限额 随机替代
		for oldkey,item := range s.kvt_map {
			s.kvt_map[key] = tmplK_item{
				kv: item.kv,
				kt: s.now,
				uid: item.uid,
			}
			delete(s.kvt_map,oldkey)
			return item.kv
		}
	}

	Uid := s.pool.Get()

	s.kvt_map[key] = tmplK_item{
		kv: Uid.Id,
		kt: s.now,
		uid: Uid,
	}

	return Uid.Id
}

func (s *tmplK) Get(key string) (isLive bool,id uintptr){
	s.RLock()
	tmp, ok := s.kvt_map[key]
	s.RUnlock()

	id = tmp.kv
	isLive = ok && s.Druation < 0 || s.now - tmp.kt <= s.Druation
	if !isLive && ok {
		s.pool.Put(tmp.uid)
		s.Lock()
		delete(s.kvt_map,key)
		s.Unlock()
	}
	return
}

func (s *tmplK) Check(key string,id uintptr) bool {
	ok,k := s.Get(key)
	return ok && k == id
}

func (s *tmplK) Buf() (int64,int) {
	return s.now,len(s.kvt_map)
}
