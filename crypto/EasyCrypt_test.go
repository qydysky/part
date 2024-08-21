package part

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func Test_EasyCrypt(t *testing.T) {
	var buf = make([]byte, 100)
	if n, e := rand.Read(buf); e != nil {
		t.Fatal(e)
	} else {
		buf = buf[:n]
	}

	if pri, pub, e := NewKey(); e != nil {
		t.Fatal(e)
	} else {
		if b, e := Encrypt(buf, pub); e != nil {
			t.Fatal(e)
		} else if msg, e := Decrypt(b, pri); e != nil {
			t.Fatal(e)
		} else if !bytes.Equal(msg, buf) {
			t.Fatal()
		}
	}
}
