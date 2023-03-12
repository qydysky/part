package part

import (
	"testing"
	"time"
)

func Test_6(t *testing.T) {
	l := New(2, "1s", "-1s")
	t0 := time.Now()
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	t0 = time.Now()
	if l.TO() || time.Until(t0.Add(time.Second)) > time.Millisecond {
		t.Fatal(time.Since(t0))
	}
}

func Test_8(t *testing.T) {
	l := New(2, "1s", "0s")
	t0 := time.Now()
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	if !l.TO() {
		t.Fatal()
	}
}

func Test_9(t *testing.T) {
	l := New(2, "1s", "5ms")
	t0 := time.Now()
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	time.Sleep(time.Millisecond * 500)
	if l.TO() || time.Until(t0.Add(time.Millisecond*500)) > time.Millisecond {
		t.Fatal()
	}
	time.Sleep(time.Millisecond * 500)
	if l.TO() || time.Until(t0.Add(time.Millisecond*505)) > time.Millisecond {
		t.Fatal()
	}
}

func Test_1(t *testing.T) {
	l := New(2, "0s", "0s")
	t0 := time.Now()
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Until(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Until(t0) > time.Millisecond {
		t.Fatal()
	}
}

func Test_2(t *testing.T) {
	l := New(2, "10s", "-1s")
	go func() {
		time.Sleep(time.Second)
		l.Close()
	}()
	t0 := time.Now()
	if l.TO() || time.Since(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Until(t0) > time.Millisecond {
		t.Fatal()
	}
	if l.TO() || time.Until(t0.Add(time.Second)) > time.Millisecond {
		t.Fatal(l.IsLive(), time.Until(t0.Add(time.Second)))
	}
}

func Test_5(t *testing.T) {
	l := New(100, "3s", "0s")
	if l.TK() != 100 {
		t.Error(`5`, l.TK())
	}
	for i := 1; i <= 50; i += 1 {
		l.TO()
	}
	if l.TK() != 50 {
		t.Error(`5`, l.TK())
	}
	time.Sleep(time.Second * 4)
	if l.PTK() != 50 {
		t.Error(`5`, l.PTK())
	}
}
