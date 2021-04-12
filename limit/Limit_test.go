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