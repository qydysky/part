package part
 
import "testing"

func Test(t *testing.T){
	var k Crypto
	if k.PubLoad() || k.PriLoad() {t.Error(`Keystatus not PublicKeyNoLoad`)}
	{
		d,_ := FileLoad(`public.pem`)
		k.GetPKIXPubKey(d)
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
}