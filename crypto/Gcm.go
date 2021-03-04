package part

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"errors"
)

type Gcm struct {
	aead *cipher.AEAD
}

func (t *Gcm) Init(key string) (error) {
	tkey,_ := hex.DecodeString(key)
	block, err := aes.NewCipher(tkey[:32])
	if err != nil {
		return err
	}

	if aesgcm, err := cipher.NewGCM(block);err != nil {
		return err
	} else {
		t.aead = &aesgcm
	}
	return nil
}

func (t *Gcm) Encrypt(source []byte) (r []byte,e error) {
	if t.aead == nil {return []byte{},errors.New(`Encrypt not init`)}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return []byte{},err
	}

	return append(nonce,(*t.aead).Seal(nil, nonce, source, nil)...),nil
}

func (t *Gcm) Decrypt(source []byte) (r []byte,e error) {
	if t.aead == nil {return []byte{},errors.New(`Decrypt not init`)}
	return (*t.aead).Open(nil, source[:12], source[12:], nil)
}