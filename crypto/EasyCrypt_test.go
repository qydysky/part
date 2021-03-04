package part

import (
	"testing"
)

func Test_EasyCrypt(t *testing.T) {

	priKey,_ := FileLoad(`private.pem`)
	pubKey,_ := FileLoad(`public.pem`)

	if sc,e := Encrypt([]byte(`asdfasdfasdf`),pubKey);e != nil {
		t.Error(e)
	} else if s,e := Decrypt(sc,priKey);e != nil {
		t.Error(e)
	} else {
		if string(s) != `asdfasdfasdf` {
			t.Error(`not match`)
		}
	}
}