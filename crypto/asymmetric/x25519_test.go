package part

import (
	"bytes"
	"crypto/rand"
	"testing"

	pcs "github.com/qydysky/part/crypto/symmetric"
)

func Test_X25519(t *testing.T) {
	var buf = make([]byte, 100)
	if n, e := rand.Read(buf); e != nil {
		t.Fatal(e)
	} else {
		buf = buf[:n]
	}

	m := X25519F
	sym := pcs.Chacha20poly1305F
	if pri, pub, e := m.NewKey(); e != nil {
		t.Fatal(e)
	} else {
		if enc, e := m.Encrypt(pub); e != nil {
			t.Fatal(e)
		} else if b, ex, e := enc(sym, buf); e != nil {
			t.Fatal()
		} else {
			b, ex = Unpack(Pack(b, ex))
			if dec, e := m.Decrypt(pri); e != nil {
				t.Fatal(e)
			} else if msg, e := dec(sym, b, ex); e != nil {
				t.Fatal(e)
			} else if !bytes.Equal(msg, buf) {
				t.Fatal()
			}
		}
	}
}
