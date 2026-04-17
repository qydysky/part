package unsafe

import (
	"reflect"
	"testing"
)

func Test5(t *testing.T) {
	var a struct {
		a string
	}

	SetField(&a, "a", "1234")

	if a.a != "1234" {
		t.Fatal()
	}

	SetField(reflect.ValueOf(&a).Elem(), "a", "12341")

	if a.a != "12341" {
		t.Fatal()
	}

	SetField(&a, 0, "123412")

	if a.a != "123412" {
		t.Fatal()
	}
}

func Benchmark(b *testing.B) {
	var a struct {
		A string
	}
	for b.Loop() {
		SetField(&a, 0, "1234")
	}
}

func Benchmark3(b *testing.B) {
	var a struct {
		A string
	}
	for b.Loop() {
		reflect.ValueOf(&a).Elem().Field(0).Set(reflect.ValueOf(""))
		// SetField(&a, "a", "1234")
	}
}
