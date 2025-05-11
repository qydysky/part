package part

import (
	"sync"
	"time"

	idpool "github.com/qydysky/part/idpool"
)

type tmplV struct {
	SumInDruation int64
	Druation      int64
	now           int64
	deleteNum     int
	pool          *idpool.Idpool[struct{}]
	kvt_map       map[uintptr]tmplV_item
	sync.RWMutex
}

type tmplV_item struct {
	kv  string
	kt  int64
	uid *idpool.Id[struct{}]
}

func New_tmplV(SumInDruation, Druation int64) *tmplV {

	s := &tmplV{
		SumInDruation: SumInDruation,
		Druation:      Druation,
		kvt_map:       make(map[uintptr]tmplV_item),
		pool:          idpool.New(func() *struct{} { return new(struct{}) }),
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			s.now = (<-ticker.C).Unix()
		}
	}()

	return s
}

func (s *tmplV) Set(contect string) (key uintptr) {

	if s.SumInDruation >= 0 && s.pool.InUse() >= s.SumInDruation { //不为无限&&达到限额 随机替代
		s.Lock()
		for key, item := range s.kvt_map {
			s.kvt_map[key] = tmplV_item{
				kv:  contect,
				kt:  s.now,
				uid: item.uid,
			}
			s.Unlock()
			return key
		}
	}

	Uid := s.pool.Get()

	s.Lock()
	s.kvt_map[Uid.Id] = tmplV_item{
		kv:  contect,
		kt:  s.now,
		uid: Uid,
	}
	s.Unlock()

	return Uid.Id
}

func (s *tmplV) Get(key uintptr) (isLive bool, contect string) {
	s.RLock()
	K, ok := s.kvt_map[key]
	s.RUnlock()
	contect = K.kv
	isLive = ok && s.Druation < 0 || s.now-K.kt <= s.Druation
	if !isLive && ok {
		s.pool.Put(K.uid)
		s.Lock()
		delete(s.kvt_map, key)
		if s.deleteNum > len(s.kvt_map) {
			s.deleteNum = 0
			go s.Tidy()
		}
		s.Unlock()
	}
	return
}

func (s *tmplV) Check(key uintptr, contect string) bool {
	ok, k := s.Get(key)
	return ok && (k == contect)
}

func (s *tmplV) Buf() (int64, int) {
	return s.now, len(s.kvt_map)
}

func (s *tmplV) Tidy() {
	tmp := make(map[uintptr]tmplV_item)
	s.Lock()
	for k, v := range s.kvt_map {
		tmp[k] = v
	}
	s.kvt_map = tmp
	s.Unlock()
}
