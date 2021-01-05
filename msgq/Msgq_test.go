package part

import (
	"time"
	"fmt"
	"testing"
	p "github.com/qydysky/part"
)

type test_item struct {
	data string
}

func Test_msgq(t *testing.T) {


	mq := New(5)
	mun := 100000
	mun_c := make(chan bool,mun)
	mun_s := make(chan bool,mun)

	var e int

	sig := mq.Sig()
	for i:=0;i<mun;i++{
		go func(){
			mun_c <- true
			data,t0 := mq.Pull(sig)
			if o,ok:=data.(string);o != `mmm`||!ok {e = 1}
			data1,_ := mq.Pull(t0)
			if o,ok:=data1.(string);o != `mm1`||!ok {e = 2}
			mun_s <- true
		}()
	}

	for len(mun_c) != mun {
		t.Log(`>`,len(mun_c))
		p.Sys().Timeoutf(1)
	}
	t.Log(`>`,len(mun_c))

	t.Log(`push mmm`)
	mq.Push(`mmm`)
	t.Log(`push mm1`)
	mq.Push(`mm1`)

	for len(mun_s) != mun {
		t.Log(`<`,len(mun_s))
		p.Sys().Timeoutf(1)
	}
	t.Log(`<`,len(mun_s))

	if e!=0 {t.Error(e)}
}

func Test_msgq2(t *testing.T) {
	mq := New(5)

	mun_c := make(chan bool,100)
	go func(){
		var (
			sig = mq.Sig()
			data interface{}
		)
		for {
			data,sig = mq.Pull(sig)
			if data.(test_item).data != `aa1` {t.Error(`1`)}
			mun_c <- true
		}
	}()	
	go func(){
		var (
			sig = mq.Sig()
			data interface{}
		)
		for {
			data,sig = mq.Pull(sig)
			if data.(test_item).data != `aa1` {t.Error(`2`)}
			mun_c <- true
		}
	}()
	go func(){
		var (
			sig = mq.Sig()
			data interface{}
		)
		for {
			data,sig = mq.Pull(sig)
			if data.(test_item).data != `aa1` {t.Error(`3`)}
			mun_c <- true
		}
	}()	
	var fin_turn = 0
	t.Log(`start`)
	time.Sleep(time.Second)
	for fin_turn < 1000000 {
		mq.Push(test_item{
			data:`aa1`,
		})
		<-mun_c
		<-mun_c
		<-mun_c
		fin_turn += 1
		fmt.Print("\r",fin_turn)
	}
	t.Log(`fin`)
}

func Test_msgq3(t *testing.T) {
	mq := New(100)

	mun_c := make(chan int,100)
	mq.Pull_tag(map[string]func(interface{})(bool){
		`A1`:func(data interface{})(bool){
			if v,ok := data.(int);ok {mun_c <- v}
			return false
		},
	})

	var fin_turn = 0
	t.Log(`start`)
	time.Sleep(time.Second)
	for fin_turn < 1000000 {
		mq.Push_tag(`A1`,fin_turn)
		if fin_turn != <-mun_c {t.Error(fin_turn)}
		fin_turn += 1
		fmt.Print("\r",fin_turn)
	}
	t.Log(`fin`)
}

func Test_msgq4(t *testing.T) {
	// mq := New(30)
	mq := New(3)//out of list

	mun_c1 := make(chan bool,100)
	mun_c2 := make(chan bool,100)
	mun_c3 := make(chan bool,100)
	mq.Pull_tag(map[string]func(interface{})(bool){
		`A1`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`1`)}
			mun_c1 <- true
			return false
		},
		`A2`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`2`)}
			mun_c2 <- true
			return false
		},
		`Error`:func(data interface{})(bool){
			if data == nil {t.Error(`out of list`)}
			return false
		},
	})
	mq.Pull_tag(map[string]func(interface{})(bool){
		`A1`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`2`)}
			mun_c3 <- true
			return false
		},
		`Error`:func(data interface{})(bool){
			if data == nil {t.Error(`out of list`)}
			return false
		},
	})

	var fin_turn = 0
	t.Log(`start`)
	time.Sleep(time.Second)
	for fin_turn < 5 {
		go mq.Push_tag(`A1`,`a11`)
		go mq.Push_tag(`A1`,`a11`)
		go mq.Push_tag(`A1`,`a11`)
		// mq.Push_tag(`A4`,`a11`)
		go mq.Push_tag(`A1`,`a11`)
		mq.Push_tag(`A1`,`a11`)
		mq.Push_tag(`A2`,`a11`)
		// mq.Push_tag(`A4`,`a11`)
		<-mun_c2		
		<-mun_c1
		// <-mun_c3
		fin_turn += 1
		fmt.Print("\r",fin_turn)
	}
	t.Log(`fin`)
}

func Test_msgq5(t *testing.T) {
	mq := New(30)

	mun_c1 := make(chan bool,100)
	mun_c2 := make(chan bool,100)
	go mq.Pull_tag(map[string]func(interface{})(bool){
		`A1`:func(data interface{})(bool){
			time.Sleep(time.Second)//will block
			return false
		},
		`A2`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`2`)}
			mun_c2 <- true
			return false
		},
		`Error`:func(data interface{})(bool){
			if data == nil {t.Error(`out of list`)}
			return false
		},
	})
	mq.Pull_tag(map[string]func(interface{})(bool){
		`A1`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`1`)}
			mun_c1 <- true
			return false
		},
		`A2`:func(data interface{})(bool){
			if v,ok := data.(string);!ok || v != `a11`{t.Error(`2`)}
			return false
		},
		`Error`:func(data interface{})(bool){
			if data == nil {t.Error(`out of list`)}
			return false
		},
	})
	
	var fin_turn = 0
	t.Log(`start`)
	time.Sleep(time.Second)
	for fin_turn < 10 {
		mq.Push_tag(`A1`,`a11`)
		mq.Push_tag(`A2`,`a11`)
		<-mun_c1
		<-mun_c2
		fin_turn += 1
		fmt.Print("\r",fin_turn)
	}
	t.Log(`fin`)
}
