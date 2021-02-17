package part

import (
	"sync"
	"testing"
)

func Test_customMap(t *testing.T) {
	var c Map
	//set
	c.Store(0, 3)
	if c.Load(0) != 3{t.Error(`1`)}
	//change
	c.Store(0, 1)
	if c.Load(0) != 1{t.Error(`2`)}
	//del
	c.Store(0, nil)
	if c.Load(0) != nil{t.Error(`3`)}
}

func Benchmark_customMap_Set(b *testing.B) {
	var c Map
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, nil)
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_customMap_Del(b *testing.B) {
	c := new(Map)
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, 0)
			w.Done()
		}(i)
	}
	w.Wait()

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Delete(index)
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_customMap_Get(b *testing.B) {
	var c Map
	var w = &sync.WaitGroup{}
	var t = b.N
	
	w.Add(t)
	b.ResetTimer()
	for i := 0; i < t; i++ {
		go func(index int) {
			c.Store(index, index)
			w.Done()
		}(i)
	}
	w.Wait()

	b.ResetTimer()

	w.Add(t)
	b.ResetTimer()
	for i := 0; i < t; i++ {
		go func(index int) {
			if c.Load(index).(int) != index {
				b.Error("q")
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_customMap_SetGet(b *testing.B) {
	var c Map
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, index)
			w.Done()
		}(i)
	}
	w.Wait()

	b.Log(c.Len())

	w.Add(b.N)
	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, index+1)
			w.Done()
		}(i)
		go func(index int) {
			if t,ok := c.Load(index).(int);!ok || t != index && t != index+1{
				b.Error(`E`, index, t)
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_syncMap_Set(b *testing.B) {
	c:=new(sync.Map)
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, nil)
			w.Done()
		}(i)
	}
	w.Wait()
 }

 func Benchmark_syncMap_Del(b *testing.B) {
	c := new(sync.Map)
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, 0)
			w.Done()
		}(i)
	}
	w.Wait()

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Delete(index)
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_syncMap_Get(b *testing.B) {
	c := new(sync.Map)
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, 0)
			w.Done()
		}(i)
	}
	w.Wait()

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			if v,ok := c.Load(index);!ok || v.(int) != 0 {b.Error("q")}
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_syncMap_SetGet(b *testing.B) {
	c := new(sync.Map)
	var w = &sync.WaitGroup{}

	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, 0)
			w.Done()
		}(i)
	}
	w.Wait()

	w.Add(b.N)
	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, 1)
			w.Done()
		}(i)
		go func(index int) {
			c.Load(index)
			w.Done()
		}(i)
	}
	w.Wait()
}
