package part

import (
	"time"

	syncmap "github.com/qydysky/part/sync"
)

type tmplKV struct {
	now     int64
	kvt_map syncmap.Map
}

type tmplKV_item struct {
	kv interface{}
	kt int64
}

// 初始化一个带超时机制的Key-Value储存器
func New_tmplKV() *tmplKV {

	s := new(tmplKV)

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			s.now = (<-ticker.C).Unix()
		}
	}()

	return s
}

// 设置Key Value Exp（有效秒数,<0永久）
func (s *tmplKV) Set(key, value interface{}, exp int64) {
	if exp >= 0 {
		exp = s.now + exp
	}
	s.kvt_map.Store(key, tmplKV_item{
		kv: value,
		kt: exp,
	})
}

// 获取Value 及是否有效
func (s *tmplKV) Get(key interface{}) (isLive bool, value interface{}) {
	tmp, ok := s.kvt_map.Load(key)

	item, _ := tmp.(tmplKV_item)
	value = item.kv

	isLive = ok && item.kt < 0 || s.now <= item.kt
	if !isLive && ok {
		s.kvt_map.Delete(key)
	}
	return
}

// 获取Value 及是否有效
func (s *tmplKV) GetV(key interface{}) (value interface{}) {
	tmp, ok := s.kvt_map.Load(key)

	item, _ := tmp.(tmplKV_item)
	value = item.kv

	isLive := ok && item.kt < 0 || s.now <= item.kt
	if !isLive && ok {
		value = nil
		s.kvt_map.Delete(key)
	}
	return
}

// 检查Key Value是否对应及有效
func (s *tmplKV) Check(key, value interface{}) bool {
	ok, v := s.Get(key)
	return ok && (v == value)
}

// 当前储存器键值数量
func (s *tmplKV) Len() (int64, int) {
	return s.now, s.kvt_map.Len()
}
