package part

import (
	"crypto/mlkem"
	"encoding/pem"

	pc "github.com/qydysky/part/crypto"
)

type Mlkem struct{}

var MlkemF pc.Asymmetric = Mlkem{}

func (t Mlkem) GetType() string {
	return `MLKEM`
}
func (t Mlkem) CheckType(b *pem.Block) (ok bool, isPriKey bool) {
	isPriKey = b.Type == t.GetType()+PriKeySuf
	if !isPriKey {
		ok = b.Type == t.GetType()+PubKeySuf
	} else {
		ok = true
	}
	return
}

func (t Mlkem) Decrypt(priKey *pem.Block) (dec pc.AsymmetricDec, e error) {
	if priKey.Type != t.GetType()+PriKeySuf {
		return nil, ErrType
	} else if d, err := mlkem.NewDecapsulationKey1024(priKey.Bytes); err != nil {
		return nil, err
	} else {
		return func(sym pc.Symmetric, b, exchangeTxt []byte) (msg []byte, e error) {
			if sharedKey, err := d.Decapsulate(exchangeTxt); err != nil {
				return nil, err
			} else {
				return sym.Decrypt(b, sharedKey)
			}
		}, nil
	}
}

func (t Mlkem) Encrypt(pubKey *pem.Block) (enc pc.AsymmetricEnc, e error) {
	if pubKey.Type != t.GetType()+PubKeySuf {
		return nil, ErrType
	} else if d, err := mlkem.NewEncapsulationKey1024(pubKey.Bytes); err != nil {
		return nil, err
	} else {
		return func(sym pc.Symmetric, msg []byte) (b, exchangeTxt []byte, e error) {
			sharedKey, ciphertext := d.Encapsulate()
			b, e = sym.Encrypt(msg, sharedKey)
			if e != nil {
				return nil, nil, e
			}
			return b, ciphertext, nil
		}, nil
	}
}

func (t Mlkem) NewKey() (pri, pub *pem.Block, e error) {
	var d *mlkem.DecapsulationKey1024
	d, e = mlkem.GenerateKey1024()
	if e != nil {
		return
	}
	return &pem.Block{
			Type:  t.GetType() + PriKeySuf,
			Bytes: d.Bytes(),
		}, &pem.Block{
			Type:  t.GetType() + PubKeySuf,
			Bytes: d.EncapsulationKey().Bytes(),
		}, nil
}
