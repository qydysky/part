package part

import (
	"errors"
	"testing"
)

func TestMain(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		t.Log(`id1`)
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		t.Log(`id2 1`)
		// some method get id2 but wrong
		return "", ErrNextMethod
	}, func() (misskey string, err error) {
		t.Log(`id2 2`)
		// some method get id2
		M.id2 = 1
		return "", nil
	})

	if e := api.Get(`id1`); e != nil {
		t.Fatal(e)
	}

	if M.id1 != 1 || M.id2 != 1 {
		t.Fatal()
	}
}

func TestMain2(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		// some method get id2 but wrong
		return "", ErrNextMethod
	}, func() (misskey string, err error) {
		// some method get id2
		M.id2 = 1
		return "", nil
	})

	for node := range api.GetTrace(`id1`).Asc() {
		t.Log(node)
	}

	if M.id1 != 1 || M.id2 != 1 {
		t.Fatal()
	}
}

func TestMain3(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		// some method get id2 but wrong
		return "", ErrNextMethod
	}, func() (misskey string, err error) {
		// some method get id2 but wrong
		return "", errors.New(`wrong`)
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if lastNode.Err.Error() != `wrong` {
		t.Fatal()
	}

	if api.Get(`id1`).Error() != `wrong` {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain4(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if lastNode.Err != ErrKeyNotReg {
		t.Fatal()
	}

	if api.Get(`id1`) != ErrKeyNotReg {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain5(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if lastNode.Err != ErrKeyNotReg {
		t.Fatal()
	}

	if api.Get(`id1`) != ErrKeyNotReg {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain6(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}
	ccc := errors.New(`custom`)
	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		// some method get id2 but wrong
		return "", ErrNextMethod.NewErr(ccc)
	}, func() (misskey string, err error) {
		// some method get id2 but id2 not get
		M.id2 = 0
		return "", nil
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
		if node.Key == `id2` && node.MethodIndex == 0 {
			if !errors.Is(node.Err, ccc) {
				t.Fatal()
			}
		}
	}

	if lastNode.Err != ErrKeyMissAgain {
		t.Fatal()
	}

	if api.Get(`id1`) != ErrKeyMissAgain {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain7(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id1`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if lastNode.Err != ErrKeyMissAgain {
		t.Fatal()
	}

	if api.Get(`id1`) != ErrKeyMissAgain {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain8(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id1`, nil
		}
		// some method get id1
		M.id2 = 1
		return "", nil
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if lastNode.Err != ErrKeyMissAgain {
		t.Fatal()
	}

	if api.Get(`id1`) != ErrKeyMissAgain {
		t.Fatal()
	}

	if M.id1 != 0 || M.id2 != 0 {
		t.Fatal(M)
	}
}

func TestMain9(t *testing.T) {
	type m struct {
		id1 int
		id2 int
	}

	M := m{
		id1: 0,
		id2: 0,
	}

	ccc := errors.New("2")

	api := NewKeyFunc().Reg(`id1`, func() bool {
		return M.id1 != 0
	}, func() (misskey string, err error) {
		if M.id2 == 0 {
			return `id2`, nil
		}
		// some method get id1
		M.id1 = 1
		return "", nil
	}).Reg(`id2`, func() bool {
		return M.id2 != 0
	}, func() (misskey string, err error) {
		return "", ErrNextMethod.New("1")
	}, func() (misskey string, err error) {
		return "", ErrNextMethod.NewErr(ccc)
	})

	lastNode := api.GetTrace(`id1`)
	for node := range lastNode.Asc() {
		t.Log(node)
	}

	if !errors.Is(lastNode.Err, ccc) {
		t.Fatal()
	}
}

func Benchmark(b *testing.B) {
	M := false
	kf := NewKeyFunc().Reg(`1`, func() bool {
		return M
	}, func() (misskey string, err error) {
		M = true
		return "", nil
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kf.Get(`1`)
	}
}
