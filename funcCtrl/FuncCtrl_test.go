package part

import (
	"testing"
	"time"
)

func Test_SkipFunc(t *testing.T) {
	var b SkipFunc
	var a = func(i int) {
		if b.NeedSkip() {
			return
		}
		defer b.UnSet()
		t.Log(i, `.`)
		time.Sleep(time.Second)
		t.Log(i, `..`)
	}
	t.Log(`just show 1 or 2 twice`)
	go a(1)
	go a(2)
	time.Sleep(5 * time.Second)
}

func Test_FlashFunc(t *testing.T) {
	var b FlashFunc
	var a = func(i int) {
		id := b.Flash()
		defer b.UnFlash()

		t.Log(i, `.`)
		time.Sleep(time.Second)
		if b.NeedExit(id) {
			return
		}
		t.Log(i, `.`)
	}
	t.Log(`show 1 or 2, then show the other twice`)
	go a(1)
	go a(2)
	time.Sleep(5 * time.Second)
}

func Test_BlockFunc(t *testing.T) {
	var b BlockFunc
	var a = func(i int) {
		b.Block()
		defer b.UnBlock()
		t.Log(i, `.`)
		time.Sleep(time.Second)
		t.Log(i, `.`)
	}
	t.Log(`show 1 and 2 twice`)
	go a(1)
	go a(2)
	time.Sleep(5 * time.Second)
}

func Test_BlockFuncN(t *testing.T) {
	var b = &BlockFuncN{
		Max: 2,
	}
	var a = func(i int) {
		defer b.UnBlock()
		b.Block()
		t.Log(i, `.`)
		time.Sleep(time.Second)
		t.Log(i, `.`)
	}
	t.Log(`show two . at one time`)
	go a(0)
	go a(1)
	go a(2)
	go a(3)
	b.None()
	b.UnNone()
	t.Log(`fin`)
}

func Test_BlockFuncNPlan(t *testing.T) {
	var b = &BlockFuncN{
		Max: 2,
	}
	var a = func(i int) {
		defer b.UnBlock()
		b.Block()
		t.Log(i, `.`)
		time.Sleep(time.Second)
		t.Log(i, `.`)
	}
	t.Log(`show two . at one time`)
	b.Plan(4)
	go a(0)
	go a(1)
	go a(2)
	go a(3)
	b.PlanDone(func() {
		time.Sleep(time.Microsecond * 10)
	})
	t.Log(`fin`)
}
