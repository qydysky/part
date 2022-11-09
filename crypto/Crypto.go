package part

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

type Crypto struct {
	pubKey *rsa.PublicKey
	priKey *rsa.PrivateKey
}

func FileLoad(path string) (data []byte, err error) {
	fileObject, e := os.OpenFile(path, os.O_RDONLY, 0644)
	if e != nil {
		err = e
		return
	}
	defer fileObject.Close()
	data, e = io.ReadAll(fileObject)
	if e != nil {
		err = e
		return
	}
	return
}

func (t *Crypto) PubLoad() bool {
	return t.pubKey != nil
}

func (t *Crypto) PriLoad() bool {
	return t.priKey != nil
}

func (t *Crypto) GetPKIXPubKey(pubPEMData []byte) (err error) {
	block, _ := pem.Decode(pubPEMData)
	if block == nil || block.Type != "PUBLIC KEY" {
		err = errors.New("failed to decode PEM block containing public key")
		return
	}

	pubI, e := x509.ParsePKIXPublicKey(block.Bytes)
	if e != nil {
		err = e
		return
	}
	t.pubKey, _ = pubI.(*rsa.PublicKey)

	return
}

func (t *Crypto) LoadPKIXPubKey(path string) (err error) {
	if d, e := FileLoad(path); e != nil {
		return e
	} else {
		err = t.GetPKIXPubKey(d)
	}
	return
}

func (t *Crypto) GetPKCS1PriKey(priPEMData []byte) (err error) {
	block, _ := pem.Decode(priPEMData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		err = errors.New("failed to decode PEM block containing private key")
		return
	}

	t.priKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)

	return
}

func (t *Crypto) LoadPKCS1PriKey(path string) (err error) {
	if d, e := FileLoad(path); e != nil {
		return e
	} else {
		err = t.GetPKCS1PriKey(d)
	}
	return
}

func (t *Crypto) GetEncrypt(sourceByte []byte) (tragetByte []byte, err error) {
	if t.pubKey == nil {
		err = errors.New(`public key not load`)
		return
	}
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, t.pubKey, sourceByte, []byte{})
}

func (t *Crypto) GetDecrypt(sourceByte []byte) (tragetByte []byte, err error) {
	if t.priKey == nil {
		err = errors.New(`private key not load`)
		return
	}
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, t.priKey, sourceByte, []byte{})
}
