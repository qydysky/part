package part

import (
	"testing"
	"time"
)

func Test_1(t *testing.T){
	l := New(10,1000,0)
	pass := 0
	for i:=0;i<1500;i+=1{
		go func(){
			if !l.TO() {pass += 1}
		}()
		time.Sleep(time.Millisecond)
	}
	if pass!=20 {t.Error(`pass != 20`)}
}

func Test_2(t *testing.T){
	l := New(10,1000,1000)
	pass := 0
	for i:=0;i<500;i+=1{
		go func(){
			if !l.TO() {pass += 1}
		}()
		time.Sleep(time.Millisecond)
	}
	if pass!=10 {t.Error(`pass != 10`,pass)}
}

func Test_3(t *testing.T){
	l := New(10,0,0)
	pass := 0
	for i:=0;i<500;i+=1{
		go func(){
			if !l.TO() {pass += 1}
		}()
		time.Sleep(time.Millisecond)
	}
	t.Log(pass)
}

func Test_4(t *testing.T){
	l := New(0,0,10)
	pass := 0
	for i:=0;i<500;i+=1{
		go func(){
			if !l.TO() {pass += 1}
		}()
		time.Sleep(time.Millisecond)
	}
	t.Log(pass)
}

func Test_5(t *testing.T){
	l := New(100,3000,0)
	t.Log(l.TK())
	for i:=1;i<=50;i+=1{
		l.TO()
	}
	if l.TK() != 50 {t.Error(`5`,l.TK())}
	time.Sleep(time.Second*3)
	if l.PTK() != 50 {t.Error(`5`,l.PTK())}
}