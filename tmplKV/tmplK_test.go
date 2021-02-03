package part

import (
	"time"
	"testing"
)

func Test_tmplK(t *testing.T) {
	s := New_tmplK(1e6, 5)
	k1 := s.Set("a")
	if !s.Check("a",k1) {t.Error(`no match1`)}
	k2 := s.Set("a")
	if s.Check("a",k1) {t.Error(`match`)}
	if !s.Check("a",k2) {t.Error(`no match2`)}
	if o,p := s.Buf();p != 1 || o - time.Now().Unix() > 1{t.Error(`sum/time no match1`)}

	time.Sleep(time.Second*time.Duration(5))

	if s.Check("a",k1) {t.Error(`no TO1`)}
	if s.Check("a",k1) {t.Error(`no TO2`)}

	if o,p := s.Buf();p != 0 || o - time.Now().Unix() > 1{t.Error(`sum/time no match2`)}
}
