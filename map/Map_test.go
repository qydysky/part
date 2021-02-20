package part

import (
	"sync"
	"testing"
)

func Test_customMap(t *testing.T) {
	var c Map
	//set
	c.Store(0, 3)
	if v,ok := c.Load(0);ok && v != 3{t.Error(`1`)}
	//change
	c.Store(0, 0)
	if v,ok := c.Load(0);ok && v != 0{t.Error(`2`)}
	//range
	c.Store(1, 1)
	c.Range(func(key,value interface{})(bool){
		t.Log(key, value)
		if key.(int) != value.(int) {t.Error(`3`)}
		return true
	})
	//del
	c.Store(0, nil)
	if v,ok := c.Load(0);ok && v != nil{t.Error(`4`)}
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
			if c.LoadV(index).(int) != index {
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

	w.Add(b.N)
	w.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(index int) {
			c.Store(index, index+1)
			w.Done()
		}(i)
		go func(index int) {
			if t,ok := c.LoadV(index).(int);!ok || t != index && t != index+1{
				b.Error(`E`, index, t)
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func Benchmark_customMap_Range(b *testing.B) {
	var c Map
	var w = &sync.WaitGroup{}
	var t = 900000//b.N

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
	c.Range(func(k,v interface{})(bool){
		if k.(int) != v.(int) {b.Error(`p`)}
		return true
	})
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

func Benchmark_syncMap_Range(b *testing.B) {
	var c sync.Map
	var w = &sync.WaitGroup{}
	var t = 900000//b.N

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
	c.Range(func(k,v interface{})(bool){
		if k.(int) != v.(int) {b.Error(`p`)}
		return true
	})
}