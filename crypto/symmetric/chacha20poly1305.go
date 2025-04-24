package part

import (
	"crypto/rand"

	pcrypto "github.com/qydysky/part/crypto"
	"golang.org/x/crypto/chacha20poly1305"
)

type Chacha20poly1305 struct{}

var Chacha20poly1305F pcrypto.Symmetric = Chacha20poly1305{}

// GetType implements part.Symmetric.
func (c Chacha20poly1305) GetType() string {
	return `CHACHA20POLY1305`
}

// Decrypt implements part.Symmetric.
func (c Chacha20poly1305) Decrypt(b []byte, key []byte) (msg []byte, e error) {
	if aead, err := chacha20poly1305.NewX(key); err != nil {
		return nil, err
	} else {
		nonce, ciphertext := b[:aead.NonceSize()], b[aead.NonceSize():]
		return aead.Open(nil, nonce, ciphertext, nil)
	}
}

// Encrypt implements part.Symmetric.
func (c Chacha20poly1305) Encrypt(msg []byte, key []byte) (b []byte, e error) {
	if aead, err := chacha20poly1305.NewX(key); err != nil {
		return nil, err
	} else {
		nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(msg)+aead.Overhead())
		if n, err := rand.Read(nonce); err != nil {
			return nil, err
		} else {
			nonce = nonce[:n]
			return aead.Seal(nonce, nonce, msg, nil), nil
		}
	}
}
