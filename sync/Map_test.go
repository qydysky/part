package part

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestA(t *testing.T) {
	m := new(MapExceeded[string, string])
	if val, loaded, _ := m.LoadOrStore("a"); val != "" || loaded {
		t.Fatal()
	}

	if val, loaded, _ := m.LoadOrStore("a"); val != "" || loaded {
		t.Fatal()
	}
}

type tmp struct {
	p int
}

func TestLS(t *testing.T) {
	var c Map
	var ls = NewLoadOrInitFunc[int](&c).SetInit(func() *int {
		var i = 1
		return &i
	})
	a0, l0 := ls.LoadOrInit(`1`)
	if l0 {
		t.Fatal()
	}
	a1, l1 := ls.LoadOrInit(`1`)
	if !l1 {
		t.Fatal()
	}
	if a0 != a1 {
		t.Fatal()
	}
}

func Test_customMap(t *testing.T) {
	var c Map
	//set
	var p1 = new(tmp)
	p1.p = 0
	c.Store(0, p1)
	if v, ok := c.Load(0); !ok || v == nil || v.(*tmp).p != 0 {
		t.Error(`1`)
	}
	//change
	c.Store(0, 0)
	if v, ok := c.Load(0); ok && v != 0 {
		t.Error(`2`)
	}
	//range
	c.Store(1, 1)
	c.Range(func(key, value interface{}) bool {
		if key.(int) != value.(int) {
			t.Error(`3`)
		}
		return true
	})
	//del
	c.Delete(0)
	if v, ok := c.Load(0); ok && v != nil {
		t.Error(`4`)
	}
	c.Delete(1)
	if v, ok := c.Load(1); ok && v != nil {
		t.Error(`6`)
	}
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
			if t, ok := c.LoadV(index).(int); !ok || t != index && t != index+1 {
				b.Error(`E`, index, t)
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func Test_Range(t *testing.T) {
	var c Map
	for i := 0; i < 100; i++ {
		c.Store(i, nil)
	}
	if c.Len() != 100 {
		t.Error("初始化错误", c.Len())
	}
	c.Range(func(k, v interface{}) bool {
		if k == 10 {
			return true
		}
		c.Delete(k)
		return true
	})
	if c.Len() != 1 {
		t.Error("Delete 无效", c.Len())
	}
}

func Test_1(t *testing.T) {
	var c Map
	c.Store("o", []string{"111", "222"})
	m := c.Copy()
	if v, ok := m.LoadV("o").([]string); !ok || len(v) != 2 || v[0] != "111" || v[1] != "222" {
		t.Fatal()
	}
	c.Delete("o")
	if v, ok := m.LoadV("o").([]string); !ok || len(v) != 2 || v[0] != "111" || v[1] != "222" {
		t.Fatal()
	}
}

func Test_2(t *testing.T) {
	var c = make(map[string]int)
	c["1"] = 1
	cc := Copy(c)
	c["1"] = 2
	if cc["1"] != 1 {
		t.Fatal()
	}
}

func Benchmark_customMap_Range(b *testing.B) {
	var c Map
	var w = &sync.WaitGroup{}
	var t = 900000 //b.N

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
	c.Range(func(k, v interface{}) bool {
		if k.(int) != v.(int) {
			b.Error(`p`)
		}
		return true
	})
}

func Benchmark_syncMap_Set(b *testing.B) {
	c := new(sync.Map)
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
			if v, ok := c.Load(index); !ok || v.(int) != 0 {
				b.Error("q")
			}
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
	var t = 900000 //b.N

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
	c.Range(func(k, v interface{}) bool {
		if k.(int) != v.(int) {
			b.Error(`p`)
		}
		return true
	})
}

func TestMapExceeded1(t *testing.T) {
	var m MapExceeded[string, *[]byte]
	var data = []byte("1")
	m.Store("1", &data, time.Second)
	if b, ok := m.Load("1"); !ok || 0 != bytes.Compare(*b, []byte("1")) {
		t.Fatal(ok, b)
	}
	time.Sleep(time.Second * 2)
	if b, ok := m.Load("1"); ok || b != nil {
		t.Fatal()
	}
}

func TestMapExceeded2(t *testing.T) {
	var m MapExceeded[string, *[]byte]
	var data = []byte("1")
	if v, loaded, f := m.LoadOrStore("1"); v != nil || loaded {
		t.Fatal()
	} else {
		f(&data, time.Second)
		if v, ok := m.Load("1"); !ok || v == nil || !bytes.Equal(data, *v) {
			t.Fatal()
		}
	}

	var w sync.WaitGroup
	w.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			v, loaded, f := m.LoadOrStore("2")
			if (!loaded && v != nil) || (loaded && !bytes.Equal(data, *v)) {
				panic("")
			}
			f(&data, time.Second)
			w.Done()
		}()
	}
	w.Wait()
}

func TestMapExceeded3(t *testing.T) {
	var m MapExceeded[string, *[]byte]
	var data = []byte("1")
	if v, loaded, f := m.LoadOrStore("1"); v != nil || loaded {
		t.Fatal()
	} else {
		f(&data, time.Second)
		if v, ok := m.Load("1"); !ok || v == nil || !bytes.Equal(data, *v) {
			t.Fatal()
		}
	}

	p := m.Copy()

	if v, ok := p.Load("1"); !ok || v == nil || !bytes.Equal(data, *v) {
		t.Fatal()
	}
}
