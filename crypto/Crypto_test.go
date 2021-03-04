package part
 
import "testing"

func Test_Crypto(t *testing.T){
	var k Crypto
	if k.PubLoad() || k.PriLoad() {t.Error(`Keystatus not PublicKeyNoLoad`)}
	{
		k.LoadPKIXPubKey(`public.pem`)
	}
	if !k.PubLoad() || k.PriLoad() {t.Error(`Keystatus not PrivateKeyNoLoad`)}
	{
		d,_ := FileLoad(`private.pem`)
		k.GetPKCS1PriKey(d)
	}
	if !k.PubLoad() || !k.PriLoad() {t.Error(`Keystatus not nil`)}
	if srcs,e := k.GetEncrypt([]byte(`1we23`));e != nil {
		t.Error(e)
	} else if des,e := k.GetDecrypt(srcs);e != nil {
		t.Error(e)
	} else {
		if s := string(des);s != `1we23` {t.Error(`not Match`,s)}
	}

	if des,e := k.GetDecrypt([]byte(`1we23`));e == nil {
		t.Error(des,e)
	}
}