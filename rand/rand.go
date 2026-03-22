package part

import (
	"bytes"
	"errors"
	"io"
	"math/rand/v2"
)

const (
	TypeNum RandType = 0b0001
	TypeHex RandType = 0b0010
	TypeLow RandType = 0b0100
	TypeUpp RandType = 0b1000
)

type RandType int

var (
	ErrSourceEmpty = errors.New("ErrSourceEmpty")
)

func randSource(typel RandType) (source []rune) {
	if typel&TypeNum == TypeNum {
		source = append(source, []rune("0123456789")...)
	}
	if typel&TypeHex == TypeHex {
		if typel&TypeNum != TypeNum {
			source = append(source, []rune("0123456789")...)
		}
		source = append(source, []rune("abcdef")...)
	}
	if typel&TypeLow == TypeLow {
		if typel&TypeHex != TypeHex {
			source = append(source, []rune("abcdef")...)
		}
		source = append(source, []rune("ghijklmnopqrstuvwxyz")...)
	}
	if typel&TypeUpp == TypeUpp {
		source = append(source, []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
	}
	if len(source) == 0 {
		panic(ErrSourceEmpty)
	}
	return
}

func Rand[T string | []byte](typel RandType, leng int) T {
	return RandGen[T](leng, randSource(typel))
}

func RandReader(typel RandType, leng int) (r io.Reader) {
	Letters := []rune(randSource(typel))
	LettersL := len(Letters)

	return rwc{
		R: func(p []byte) (i int, err error) {
			if leng == 0 {
				return 0, io.EOF
			}
			for i = 0; leng > 0 && i < len(p); i, leng = i+1, leng-1 {
				p[i] = byte(Letters[int(rand.Uint64N(uint64(LettersL)))])
			}
			return
		},
	}
}

func RandGen[T string | []byte](leng int, source []rune) T {
	sourceL := len(source)
	var bb bytes.Buffer
	bb.Grow(leng)
	for i := 0; i < leng; i++ {
		bb.WriteRune(source[int(rand.Uint64N(uint64(sourceL)))])
	}
	switch any(new(T)).(type) {
	case *string:
		return T(bb.String())
	case *[]byte:
		return T(bb.Bytes())
	}
	return *new(T)
}

type rwc struct {
	R func(p []byte) (n int, err error)
	W func(p []byte) (n int, err error)
	C func() error
}

func (t rwc) Write(p []byte) (n int, err error) {
	if t.W != nil {
		return t.W(p)
	}
	return 0, nil
}
func (t rwc) Read(p []byte) (n int, err error) {
	if t.R != nil {
		return t.R(p)
	}
	return 0, nil
}
func (t rwc) Close() error {
	if t.C != nil {
		return t.C()
	}
	return nil
}
