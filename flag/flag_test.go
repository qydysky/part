package part

import (
	"flag"
	"testing"
	"time"
)

func init() {
	flag.String("sss", "", "")
	flag.Int("i32", 0, "")
	flag.Int("f34", 0, "")
	flag.Bool("btrue", false, "")
	flag.Duration("d1m", time.Second, "")
	testing.Init()
	flag.Parse()
}

func TestMain(t *testing.T) {
	if Lookup("sss", "") != "ss" {
		t.Fatal()
	}
	if Lookup("s", "") != "" {
		t.Fatal()
	}
	if Lookup("i32", 0) != 32 {
		t.Fatal()
	}
	if Lookup("f34", 0.0) != 34 {
		t.Fatal()
	}
	if !Lookup("btrue", false) {
		t.Fatal()
	}
	if Lookup("d1m", time.Second).Seconds() != 60 {
		t.Fatal()
	}
}
