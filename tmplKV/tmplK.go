package part

import (
	"time"
	"container/list"
	syncmap "github.com/qydysky/part/map"
	idpool "github.com/qydysky/part/idpool"
)

type tmplK struct {
	SumInDruation int64
	Druation int64
	now int64
	pool *idpool.Idpool
	kvt_map syncmap.Map
	slowBackList *list.List
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
		pool:idpool.New(),
		slowBackList:list.New(),
	}
	go func(){
		ticker := time.NewTicker(time.Second)
		for{
			s.now = (<- ticker.C).Unix()
		}
	}()

	return s
}

func (s *tmplK) Set(key interface{}) (id uintptr) {
	
	if tmp, oks := s.kvt_map.LoadV(key).(tmplK_item);oks {
		s.free(tmp.uid)
	} else if s.SumInDruation >= 0 && s.freeLen() >= s.SumInDruation{//不为无限&&达到限额 随机替代
		s.kvt_map.Range(func(oldkey,item interface{})(bool){
			id = item.(tmplK_item).kv
			s.kvt_map.Store(key, tmplK_item{
				kv: id,
				kt: s.now,
				uid: item.(tmplK_item).uid,
			})
			s.kvt_map.Delete(oldkey)
			return false
		})
		return
	}

	Uid := s.pool.Get()

	s.kvt_map.Store(key, tmplK_item{
		kv: Uid.Id,
		kt: s.now,
		uid: Uid,
	})

	return Uid.Id
}

func (s *tmplK) Get(key interface{}) (isLive bool,id uintptr){
	tmp, ok := s.kvt_map.Load(key)

	item,_ := tmp.(tmplK_item)
	id = item.kv

	isLive = ok && s.Druation < 0 || s.now - item.kt <= s.Druation
	if !isLive && ok {
		s.free(item.uid)
		s.kvt_map.Delete(key)
	}
	return
}

func (s *tmplK) Check(key interface{},id uintptr) bool {
	ok,k := s.Get(key)
	return ok && (k == id)
}

func (s *tmplK) Len() (int64,int) {
	return s.now,s.kvt_map.Len()
}

func (s *tmplK) freeLen() (int64) {
	return int64(int(s.pool.Len()) + s.slowBackList.Len())
}

func (s *tmplK) free(i *idpool.Id) {
	s.slowBackList.PushBack(i)
	if s.freeLen() > s.SumInDruation {
		if el := s.slowBackList.Front();el != nil && el.Value != nil{
			e := s.slowBackList.Remove(el)
			s.pool.Put(e.(*idpool.Id))
		}
	}
}