package part

import (
	"github.com/thedevsaddam/gojsonq"
)

type json struct {}

func Json() (*json) {return &json{}}

func (*json) GetValFrom(Source interface{},key string)interface {}{
	var jq *gojsonq.JSONQ
	switch Source.(type) {
    case string:
		if Checkfile().IsExist(Source.(string)) {
			jq = gojsonq.New().File(Source.(string))
		}else{
			jq = gojsonq.New().FromString(Source.(string))
		}
	default:
        jq = gojsonq.New().FromInterface(Source)
    }
	return jq.Find(key)
}

func (this *json) GetMultiValFrom(Source interface{},key []string) []interface{}{
	var jq *gojsonq.JSONQ
	switch Source.(type) {
    case string:
		if Checkfile().IsExist(Source.(string)) {
			jq = gojsonq.New().File(Source.(string))
		}else{
			jq = gojsonq.New().FromString(Source.(string))
		}
	default:
        jq = gojsonq.New().FromInterface(Source)
    }

	var returnVal []interface{}
	for _,i := range key {
		jq.Reset()
		returnVal = append(returnVal,jq.Find(i))
	}

	return returnVal
}