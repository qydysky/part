package part

import (
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	s := Easy_boot()
	t.Log(`http://`+s.Server.Addr)
	time.Sleep(time.Second*time.Duration(100))
}