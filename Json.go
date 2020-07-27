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