package part

import (
	"testing"
)

func Test_GCM(t *testing.T){
	var s []byte
	{
		var gcm Gcm
		if e := gcm.Init(`sdifw023jfa;oo`);e != nil{
			t.Error(e)
		} else if s,e = gcm.Encrypt([]byte(`12345`));e != nil{
			t.Error(e)
		}
	}
	
	var gcm Gcm
	if e := gcm.Init(`sdifw023jfa;oo`);e != nil{
		t.Error(e)
	} else if ss,e := gcm.Decrypt(s);e != nil{
		t.Error(e)
	} else {
		if string(ss) != `12345` {t.Error(string(ss))}
	}
}