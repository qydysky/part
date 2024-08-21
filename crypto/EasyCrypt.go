package part

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/pem"
	"errors"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	pemType    = `ECDH`
	ErrPemType = errors.New(`ErrPemType`)
)

func NewKey() (pri, pub []byte, e error) {
	if p1, e := ecdh.X25519().GenerateKey(rand.Reader); e != nil {
		return nil, nil, e
	} else {
		return pem.EncodeToMemory(&pem.Block{
				Type:  pemType + ` PRIVATE KEY`,
				Bytes: p1.Bytes(),
			}), pem.EncodeToMemory(&pem.Block{
				Type:  pemType + ` PUBLIC KEY`,
				Bytes: p1.PublicKey().Bytes(),
			}), nil
	}
}

func Encrypt(msg, pubKey []byte) (b []byte, e error) {
	c := ecdh.X25519()
	var (
		p1   *ecdh.PrivateKey
		q1   *ecdh.PublicKey
		q2   *ecdh.PublicKey
		key  []byte
		aead cipher.AEAD
	)
	if p1, e = c.GenerateKey(rand.Reader); e != nil {
		return
	}
	q1 = p1.PublicKey()

	if pb, _ := pem.Decode(pubKey); pb.Type != pemType+` PUBLIC KEY` {
		e = ErrPemType
		return
	} else if q2, e = ecdh.X25519().NewPublicKey(pb.Bytes); e != nil {
		return
	}

	if key, e = p1.ECDH(q2); e != nil {
		return
	}

	if aead, e = chacha20poly1305.NewX(key); e != nil {
		return
	}
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(msg)+aead.Overhead())
	if n, err := rand.Read(nonce); err != nil {
		return nil, err
	} else {
		nonce = nonce[:n]
	}
	b = q1.Bytes()
	b = append(itob32(int32(len(b))), b...)
	b = append(b, aead.Seal(nonce, nonce, msg, nil)...)
	return
}

func Decrypt(b, priKey []byte) (msg []byte, e error) {
	var (
		q1     *ecdh.PublicKey
		p2     *ecdh.PrivateKey
		pemLen = int(btoi32(b[:4]))
		key    []byte
		aead   cipher.AEAD
	)
	if pb, _ := pem.Decode(priKey); pb.Type != pemType+` PRIVATE KEY` {
		e = ErrPemType
		return
	} else if p2, e = ecdh.X25519().NewPrivateKey(pb.Bytes); e != nil {
		return
	} else if q1, e = ecdh.X25519().NewPublicKey(b[4 : 4+pemLen]); e != nil {
		return
	} else if key, e = p2.ECDH(q1); e != nil {
		return
	} else {
		if aead, e = chacha20poly1305.NewX(key); e != nil {
			return
		}
		nonce, ciphertext := b[4+pemLen:4+pemLen+aead.NonceSize()], b[4+pemLen+aead.NonceSize():]
		if msg, e = aead.Open(nil, nonce, ciphertext, nil); e != nil {
			return
		}
	}
	return
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
