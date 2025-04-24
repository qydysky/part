package part

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/pem"

	pc "github.com/qydysky/part/crypto"
)

var X25519F pc.Asymmetric = X25519{}

type X25519 struct{}

// CheckType implements part.Asymmetric.
func (t X25519) CheckType(b *pem.Block) (ok bool, isPriKey bool) {
	isPriKey = b.Type == t.GetType()+PriKeySuf
	if !isPriKey {
		ok = b.Type == t.GetType()+PubKeySuf
	} else {
		ok = true
	}
	return
}

func (t X25519) GetType() string {
	return `ECDH` // 为了保证向后兼容，此处为ECDH
}

func (t X25519) Decrypt(priKey *pem.Block) (dec pc.AsymmetricDec, e error) {
	if priKey.Type != t.GetType()+PriKeySuf {
		return nil, ErrType
	}

	var p2 *ecdh.PrivateKey
	if p2, e = ecdh.X25519().NewPrivateKey(priKey.Bytes); e != nil {
		return
	}

	return func(sym pc.Symmetric, b, exchangeTxt []byte) (msg []byte, e error) {
		if q1, err := ecdh.X25519().NewPublicKey(exchangeTxt); err != nil {
			return nil, err
		} else if key, err := p2.ECDH(q1); err != nil {
			return nil, err
		} else {
			return sym.Decrypt(b, key)
		}
	}, nil
}

func (t X25519) Encrypt(pubKey *pem.Block) (enc pc.AsymmetricEnc, e error) {
	if pubKey.Type != t.GetType()+PubKeySuf {
		return nil, ErrType
	}

	var (
		p1  *ecdh.PrivateKey
		q1  *ecdh.PublicKey
		q2  *ecdh.PublicKey
		key []byte
	)
	if p1, e = ecdh.X25519().GenerateKey(rand.Reader); e != nil {
		return
	}
	q1 = p1.PublicKey()
	if q2, e = ecdh.X25519().NewPublicKey(pubKey.Bytes); e != nil {
		return
	} else if key, e = p1.ECDH(q2); e != nil {
		return
	} else {
		return func(sym pc.Symmetric, msg []byte) (b []byte, exchangeTxt []byte, e error) {
			b, e = sym.Encrypt(msg, key)
			exchangeTxt = q1.Bytes()
			return
		}, nil
	}
}

func (t X25519) NewKey() (pri, pub *pem.Block, e error) {
	if d, e := ecdh.X25519().GenerateKey(rand.Reader); e != nil {
		return nil, nil, e
	} else {
		return &pem.Block{
				Type:  t.GetType() + PriKeySuf,
				Bytes: d.Bytes(),
			}, &pem.Block{
				Type:  t.GetType() + PubKeySuf,
				Bytes: d.PublicKey().Bytes(),
			}, nil
	}
}
