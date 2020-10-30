package part

import (
	"os"
	"github.com/thedevsaddam/gojsonq/v2"
)

type Map struct {
	B map[string]interface{}
}

func New() *Map {
	b := new(Map)
	b.B = make(map[string]interface{})
	return b
} 

func (i *Map) Get(key string) (interface{},bool) {
	v,ok := i.B[key]
	return v,ok
}

func (i *Map) Set(key string,val interface{}) bool {
	switch val.(type) {
	case string,bool,int,float64:
		i.B[key] = val
		return true
	default:
	}
	return false
}

func (i *Map) Save(Source string) error {
	js := gojsonq.New().FromInterface(i.B)
	fileObj,err := os.OpenFile(Source,os.O_RDWR|os.O_EXCL,0644)
	if err != nil {return err}
	defer fileObj.Close()
	js.Writer(fileObj)
	return nil
}

func (i *Map) Load(Source string) {
	if b := gojsonq.New().File(Source).Get();b != nil{
		i.B = b.(map[string]interface {})
	}
}