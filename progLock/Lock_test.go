package part

import (
	"testing"
	"time"
)

func Test(t *testing.T) {
	l := New()
	t.Log(l.Start("111","222"))
	t.Log(New().Start())
	time.Sleep(time.Second)
	t.Log(l.Stop())
	t.Log(l.Stop())
}