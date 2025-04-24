package part

import (
	"encoding/pem"
)

type Asymmetric interface {
	GetType() string
	CheckType(b *pem.Block) (ok bool, isPriKey bool)
	NewKey() (pri, pub *pem.Block, e error)
	Encrypt(pubKey *pem.Block) (enc AsymmetricEnc, e error)
	Decrypt(priKey *pem.Block) (dec AsymmetricDec, e error)
}

// func(sym Symmetric, msg []byte) (b, exchangeTxt []byte, e error)
type AsymmetricEnc func(sym Symmetric, msg []byte) (b, exchangeTxt []byte, e error)

// func(sym Symmetric, b, exchangeTxt []byte) (msg []byte, e error)
type AsymmetricDec func(sym Symmetric, b, exchangeTxt []byte) (msg []byte, e error)

type Symmetric interface {
	GetType() string
	Encrypt(msg, key []byte) (b []byte, e error)
	Decrypt(b, key []byte) (msg []byte, e error)
}
