package part

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

// 9053 ns/op               8 B/op          1 allocs/op
func Benchmark_2(b *testing.B) {
	type g struct {
		A123 int
		Asf  string
	}
	a := g{}

	reader := NewBufReader()
	decoder := json.NewDecoder(reader)
	defer reader.Close()
	data := []byte(`{"A123":123,"asf":"1"}`)

	for b.Loop() {
		reader.Put(data)
		decoder.Decode(&a)
	}
}

// 6388 ns/op             224 B/op          4 allocs/op
func Benchmark_1(b *testing.B) {
	type g struct {
		A123 int
		Asf  string
	}
	a := g{}
	data := []byte(`{"A123":123,"asf":"1"}`)
	for b.Loop() {
		json.Unmarshal(data, &a)
	}
}

func Test_NewBufReader(t *testing.T) {
	type g struct {
		A123 int
		Asf  string
	}
	a := g{}

	reader := NewBufReader()
	defer reader.Close()

	reader.Put([]byte(`{"A123":123,"asf":"1"}`))
	decoder := json.NewDecoder(reader)
	if e := decoder.Decode(&a); e != nil || a.A123 != 123 || a.Asf != "1" {
		t.Fatal(e)
	}

	var wg sync.WaitGroup
	var n1, n3 time.Time

	wg.Go(func() {
		time.Sleep(time.Millisecond * 200)
		reader.Put([]byte(`{"A123":123,"asf":"1"}`))
		n1 = time.Now()
	})

	if e := decoder.Decode(&a); e != nil || a.A123 != 123 || a.Asf != "1" {
		t.Fatal(e)
	}
	n3 = time.Now()

	wg.Wait()
	if n3.Before(n1) {
		t.Fatal()
	}
}

func Test_iopipe(t *testing.T) {
	pipe := NewPipe()
	ctx, cancle := context.WithCancelCause(context.Background())
	pipe.WithCtx(ctx)
	cancle(io.ErrNoProgress)
	fmt.Println(pipe.Write([]byte{}))
}

func Test_CopyIO(t *testing.T) {
	var s = make([]byte, 1<<17+2)
	s[1<<17-1] = '1'
	s[1<<17] = '2'
	s[1<<17+1] = '3'

	var w = &bytes.Buffer{}

	if e := Copy(bytes.NewReader(s), w, CopyConfig{1<<17 + 1, 1, 0, 0, 0}); e != nil || w.Len() != 1<<17+1 || w.Bytes()[1<<17-1] != '1' || w.Bytes()[1<<17] != '2' {
		t.Fatal(e)
	}
}

func Test_rwc(t *testing.T) {
	rwc := RWC{R: func(p []byte) (n int, err error) { return 1, nil }}
	rwc.Close()
}

func Test_RW2Chan(t *testing.T) {
	{
		r, w := io.Pipe()
		_, rw := RW2Chan(nil, w)

		go func() {
			rw <- []byte{0x01}
		}()
		buf := make([]byte, 1<<16)
		n, _ := r.Read(buf)
		if buf[:n][0] != 1 {
			t.Error(`no`)
		}
	}

	{
		r, w := io.Pipe()
		rc, _ := RW2Chan(r, nil)

		go func() {
			w.Write([]byte{0x09})
		}()
		if b := <-rc; b[0] != 9 {
			t.Error(`no2`)
		}
	}

	{
		r, w := io.Pipe()
		rc, rw := RW2Chan(r, w)

		go func() {
			rw <- []byte{0x07}
		}()
		if b := <-rc; b[0] != 7 {
			t.Error(`no3`)
		}
	}
}

func Test_readall(t *testing.T) {
	var buf = []byte{}
	result, e := ReadAll(bytes.NewReader([]byte{0x01, 0x02, 0x03}), buf)
	if e != nil || !bytes.Equal(result, []byte{0x01, 0x02, 0x03}) {
		t.Fatal()
	}
}

// 4248350               281.0 ns/op            16 B/op          1 allocs/op
func Benchmark_readall(b *testing.B) {
	var buf = []byte{}
	var data = []byte{0x01, 0x02, 0x03}
	r := bytes.NewReader(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadAll(r, buf)
		r.Reset(data)
	}
}

// 2806576               424.2 ns/op           512 B/op          1 allocs/op
func Benchmark_readall1(b *testing.B) {
	var data = []byte{0x01, 0x02, 0x03}
	r := bytes.NewReader(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		io.ReadAll(r)
		r.Reset(data)
	}
}

func Test_CacheWrite(t *testing.T) {
	r, w := io.Pipe()
	rc, _ := RW2Chan(r, nil)
	go func() {
		time.Sleep(time.Millisecond * 500)
		b := <-rc
		if !bytes.Equal(b, []byte("123")) {
			t.Fatal()
		}
	}()
	writer := NewCacheWriter(w, 1)
	if n, err := writer.Write([]byte("123")); n != 3 || err != nil {
		t.Fatal()
	}
	if _, err := writer.Write([]byte("123")); err == nil {
		t.Fatal()
	}
	time.Sleep(time.Second)
}

func BenchmarkCache(b *testing.B) {
	writer := NewCacheWriter(io.Discard, 2000)
	tmp := []byte("1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := writer.Write(tmp); err != nil {
			b.Fatal(err)
		}
	}
}
