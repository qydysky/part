package part

import (
	"encoding/hex"
	"testing"
)

func Test_GCM(t *testing.T) {
	key := hex.EncodeToString([]byte("12345678901234567890123456789012"))
	var s []byte
	{
		var gcm Gcm
		if e := gcm.Init(key); e != nil {
			t.Error(e)
		} else if s, e = gcm.Encrypt([]byte(`12345`)); e != nil {
			t.Error(e)
		}
	}

	var gcm Gcm
	if e := gcm.Init(key); e != nil {
		t.Error(e)
	} else if ss, e := gcm.Decrypt(s); e != nil {
		t.Error(e)
	} else {
		if string(ss) != `12345` {
			t.Error(string(ss))
		}
	}
}
