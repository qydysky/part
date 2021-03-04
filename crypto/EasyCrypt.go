package part

import (
	"bytes"
	"errors"
	p "github.com/qydysky/part"
)

func Encrypt(source,pubKey []byte) ([]byte,error) {
	var c Crypto
	if e := c.GetPKIXPubKey(pubKey);e != nil{return []byte{},e}

	key := p.Stringf().Rand(2,32)

	var g Gcm
	if e := g.Init(key);e != nil {return []byte{},e}

	if S_body,e := g.Encrypt(source);e != nil{
		return []byte{},e
	} else if S_key,e := c.GetEncrypt([]byte(key));e != nil {
		return []byte{},e
	} else {
		return append(S_key,append([]byte(`  `),S_body...)...),nil
	}
}

func Decrypt(source,priKey []byte) ([]byte,error) {
	var loc = -1
	if loc = bytes.Index(source, []byte(`  `));loc == -1{
		return []byte{},errors.New(`not easyCrypt type`)
	}

	S_key := source[:loc]
	S_body := source[loc+2:]

	var c Crypto
	if e := c.GetPKCS1PriKey(priKey);e != nil{return []byte{},e}

	var g Gcm
	if key,e := c.GetDecrypt(S_key);e != nil {
		return []byte{},e
	} else if e := g.Init(string(key));e != nil {
		return []byte{},e
	} else if body,e := g.Decrypt(S_body);e != nil{
		return []byte{},e
	} else {
		return body,nil
	}
}