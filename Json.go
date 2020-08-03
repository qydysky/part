package part

import (
	"github.com/thedevsaddam/gojsonq"
)

type json struct {}

func Json() (*json) {return &json{}}

func (*json) GetValFrom(file,key string)interface {}{
	var jq *gojsonq.JSONQ
	if Checkfile().IsExist(file) {
		jq = gojsonq.New().File(file)
	}else{
		jq = gojsonq.New().FromString(file)
	}
	return jq.Find(key)
}

func (this *json) GetMultiValFrom(file string,key []string) []interface{}{
	var jq *gojsonq.JSONQ
	if Checkfile().IsExist(file) {
		jq = gojsonq.New().File(file)
	}else{
		jq = gojsonq.New().FromString(file)
	}

	var returnVal []interface{}
	for _,i := range key {
		jq.Reset()
		returnVal = append(returnVal,jq.Find(i))
	}

	return returnVal
}