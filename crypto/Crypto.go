package main

import (
	"os"
	"io/ioutil"
	"errors"
	"crypto/rand"
	"crypto/sha256"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

var (
	PublicKeyNoLoad = errors.New(`PublicKeyNoLoad`)
	PrivateKeyNoLoad = errors.New(`PrivateKeyNoLoad`)
)

type Crypto struct {
	pubKey *rsa.PublicKey
	priKey *rsa.PrivateKey
}

func FileLoad(path string) (data []byte, err error) {
	fileObject,e := os.OpenFile(path, os.O_RDONLY, 0644)
	if e != nil {
		err = e
		return
	}
	defer fileObject.Close()
	data,e = ioutil.ReadAll(fileObject)
	if e != nil {
		err = e
		return
	}
	return
}

func (t *Crypto) KeyStatus() (error) {
	if t.pubKey == nil {
		return PublicKeyNoLoad
	}
	if t.priKey == nil {
		return PrivateKeyNoLoad
	}
	return nil
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
	t.pubKey = pubI.(*rsa.PublicKey)

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