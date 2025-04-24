package part

import (
	"encoding/pem"
	"errors"

	pc "github.com/qydysky/part/crypto"
)

var (
	PriKeySuf string = ` PRIVATE KEY`
	PubKeySuf string = ` PUBLIC KEY`
	ErrType   error  = errors.New(`ErrType`)
)

func ChoseAsymmetricByPem(b *pem.Block) pc.Asymmetric {
	if ok, _ := X25519F.CheckType(b); ok {
		return X25519F
	} else if ok, _ := MlkemF.CheckType(b); ok {
		return MlkemF
	} else {
		return nil
	}
}

func Pack(b, exchangeTxt []byte) (a []byte) {
	buf := make([]byte, 4+len(exchangeTxt)+len(b))
	n := copy(buf, itob32(int32(len(exchangeTxt))))
	n += copy(buf[n:], exchangeTxt)
	copy(buf[n:], b)
	return buf
}

func Unpack(a []byte) (b, exchangeTxt []byte) {
	exL := btoi32(a[:4])
	return a[4+exL:], a[4 : 4+exL]
}

func itob32(v int32) []byte {
	//binary.BigEndian.PutUint32
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}

func btoi32(bu []byte) uint32 {
	return uint32(bu[3]) | uint32(bu[2])<<8 | uint32(bu[1])<<16 | uint32(bu[0])<<24
}
