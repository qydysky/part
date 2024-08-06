package part

// import (
// 	"io"
// 	"os"
// 	"errors"
// 	"strings"
// 	goej "encoding/json"
// 	"github.com/thedevsaddam/gojsonq/v2"
// )

// type json struct {}

// func Json() (*json) {return &json{}}

// func (*json) Check(Source interface{}) error {
// 	var jq *goej.Decoder
// 	switch Source.(type) {
//     case string:
// 		if Checkfile().IsExist(Source.(string)) {
// 			fileObj,err := os.OpenFile(Source.(string),os.O_RDONLY,0644)
// 			if err != nil {
// 				return err
// 			}
// 			defer fileObj.Close()
// 			jq = goej.NewDecoder(fileObj)
// 		}else{
// 			jq = goej.NewDecoder(strings.NewReader(Source.(string)))
// 		}
// 	default:
//         return errors.New("json type != string")
// 	}

// 	for {
// 		_, err := jq.Token()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (*json) GetValFrom(Source interface{},key string)interface {}{
// 	var jq *gojsonq.JSONQ
// 	switch Source.(type) {
//     case string:
// 		if Checkfile().IsExist(Source.(string)) {
// 			jq = gojsonq.New().File(Source.(string))
// 		}else{
// 			jq = gojsonq.New().FromString(Source.(string))
// 		}
// 	default:
//         jq = gojsonq.New().FromInterface(Source)
//     }
// 	return jq.More().Find(key)
// }

// func (*json) GetValFromS(Source string,key string)interface {}{
// 	var jq *gojsonq.JSONQ
// 	jq = gojsonq.New().FromString(Source)
// 	return jq.More().Find(key)
// }

// func (*json) GetArrayFrom(Source interface{},key string)[]interface {}{
// 	var jq *gojsonq.JSONQ
// 	switch Source.(type) {
//     case string:
// 		jq = gojsonq.New().FromString(Source.(string))
// 	default:
//         jq = gojsonq.New().FromInterface(Source)
//     }
// 	return jq.Pluck(key).([]interface{})
// }

// func (this *json) GetMultiValFrom(Source interface{},key []string) []interface{}{
// 	var jq *gojsonq.JSONQ
// 	switch Source.(type) {
//     case string:
// 		if Checkfile().IsExist(Source.(string)) {
// 			jq = gojsonq.New().File(Source.(string))
// 		}else{
// 			jq = gojsonq.New().FromString(Source.(string))
// 		}
// 	default:
//         jq = gojsonq.New().FromInterface(Source)
//     }

// 	var returnVal []interface{}
// 	for _,i := range key {
// 		jq.Reset()
// 		returnVal = append(returnVal,jq.More().Find(i))
// 	}

// 	return returnVal
// }
