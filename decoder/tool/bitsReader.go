package tool

type BitsReader struct {
	Data         []byte
	len          uint
	readedByte   uint
	readedInByte uint8
}

func NewBitsReader(Data []byte) *BitsReader {
	return &BitsReader{Data: Data, len: uint(len(Data))}
}

func U[T uint8 | uint16 | uint32 | uint64](t *BitsReader, n uint8) (r T) {
	r = T(t.Data[t.readedByte])
	r &= 0b11111111 >> t.readedInByte
	if 8-t.readedInByte > n {
		r = r >> (8 - t.readedInByte - n)
		t.readedInByte += n
	} else if 8-t.readedInByte == n {
		t.readedInByte = 0
		t.readedByte += 1
	} else {
		r = r << (n - (8 - t.readedInByte))
		t.readedInByte = 0
		t.readedByte += 1
		r |= U[T](t, (n - (8 - t.readedInByte)))
	}
	return
}

// unsigned integer Exp-Golomb-coded syntax element with the left bit first
func UE[T uint8 | uint16 | uint32 | uint64](t *BitsReader) (r T) {
	var move uint8
	for U[uint8](t, 1) == 0 {
		move += 1
	}
	return (U[T](t, move) | 1<<(move)) - 1
}

// signed integer Exp-Golomb-coded syntax element with the left bit first
func SE[T int8 | int16 | int32 | int64](t *BitsReader) (r T) {
	var move uint8
	for U[uint8](t, 1) == 0 {
		move += 1
	}
	switch any(r).(type) {
	case int8:
		r = T(U[uint8](t, move-1) | 1<<(move-1))
	case int16:
		r = T(U[uint16](t, move-1) | 1<<(move-1))
	case int32:
		r = T(U[uint32](t, move-1) | 1<<(move-1))
	case int64:
		r = T(U[uint64](t, move-1) | 1<<(move-1))
	}
	if U[uint8](t, 1) == 0 {
		r = -r
	}
	return
}
