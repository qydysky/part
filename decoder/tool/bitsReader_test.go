package tool

import (
	"testing"
)

func Benchmark_u(b *testing.B) {
	r := NewBitsReader([]byte{0x03, 0x03})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.readedByte = 0
		r.readedInByte = 0
		U[uint16](r, 16)
	}
}

func Benchmark_u1(b *testing.B) {
	r := []byte{0x03, 0x03}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Btoui16(r, 0)
	}
}

func Test_u(t *testing.T) {
	r := NewBitsReader([]byte{0x03, 0x03})
	t.Logf("%d %d", U[uint16](r, 16), Btoui16([]byte{0x03, 0x03}, 0))
}

func Test_ue(t *testing.T) {
	r := NewBitsReader([]byte{0b01101100})
	if r := UE[uint8](r); r != 2 {
		t.Fatal(r)
	}
	if r := UE[uint8](r); r != 2 {
		t.Fatal(r)
	}
}

func Test_se(t *testing.T) {
	r := NewBitsReader([]byte{0b00101001, 0b00100000})
	if r := SE[int8](r); r != 2 {
		t.Fatal()
	}
	if r := SE[int8](r); r != -2 {
		t.Fatal()
	}
}

func Test_mix(t *testing.T) {
	r := NewBitsReader([]byte{0b00101011, 0b00100101, 0b0})
	if r := SE[int8](r); r != 2 {
		t.Fatal()
	}
	if r := UE[uint8](r); r != 2 {
		t.Fatal(r)
	}
	if r := SE[int8](r); r != -2 {
		t.Fatal(r)
	}
	if r := U[uint8](r, 1); r != 1 {
		t.Fatal(r)
	}
	if r := UE[uint8](r); r != 1 {
		t.Fatal(r)
	}
}
