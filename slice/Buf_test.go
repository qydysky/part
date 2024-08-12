package part

import (
	"runtime"
	"testing"
	"time"
)

func TestBuf(t *testing.T) {
	bu := NewBufs[byte]()
	allocs(bu.Get())
	runtime.GC()
	time.Sleep(time.Second)
	if bu.CacheCount() != 1 {
		t.Fatal()
	}
	b := allocs(bu.Get())
	runtime.GC()
	time.Sleep(time.Second)
	if bu.CacheCount() != 0 {
		t.Fatal()
	}
	allocs(bu.Get())
	runtime.GC()
	time.Sleep(time.Second)
	t.Log(b, bu.CacheCount())
	if bu.CacheCount() != 1 {
		t.Fatal()
	}
}

func allocs(b []byte) []byte {
	return append(b, 0x01)
}
