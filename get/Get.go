package part

import (
	"strings"
	"errors"
	"net/http"
	reqf "github.com/qydysky/part/reqf"
)

type get struct {
	body []byte
	
	Response *http.Response
	RS []string
	Err error
}

func Get(r reqf.Rval) (o *get){
	o = new(get)

	if r.Url == "" {o.Err = errors.New("url == nil");return}

	R := reqf.Req()
	o.Err = R.Reqf(r)
	(*o).body = R.Respon
	(*o).Response = R.Response
	
	return
}

func (i *get) S(stratS,endS string, startI,lenI int) (o *get) {
	o = i
	var tmp string
	tmp,o.Err = SS(string(o.body), stratS, endS, startI, lenI)
	if o.Err != nil {return}
	o.RS = []string{tmp}
	return
}

func (i *get) S2(stratS,endS string) (o *get) {
	o = i
	o.RS,o.Err = SS2(string(o.body), stratS, endS)
	if o.Err != nil {return}
	return
}

func SS2(source,stratS,endS string) (return_val []string,last_err error) {
	if source == `` {last_err = errors.New("ss2:no source");return}
	if stratS == `` {last_err = errors.New("ss2:no stratS");return}

	return_val = strings.Split(source,stratS)[1:]
	if len(return_val) == 0 {last_err = errors.New("ss2:no found");return}
	if endS == `` {return}
	for k,v := range return_val {
		first_index := strings.Index(v,endS)
		if first_index == -1 {continue}
		return_val[k] = string([]rune(v)[:first_index])
	}
	return
}

func SS(source,stratS,endS string, startI,lenI int) (string,error) {
	if stratS == "" && startI == 0 {return "", errors.New("no symbol to start")}
	if endS == "" && lenI == 0 {return "", errors.New("no symbol to stop")}

	var ts,te int

	if stratS != "" {
		if tmp := strings.Index(source, stratS);tmp != -1{ts = tmp + len(stratS)}
	} else if startI != 0 {
		if startI < len(source){ts = startI}
	}

	if ts == 0 {return "", errors.New("no start symbol "+ stratS +" in " + source)}

	if endS != "" {
		if tmp := strings.Index(source[ts:], endS);tmp != -1{te = ts + tmp}
	} else if lenI != 0 {
		if startI + lenI < len(source){te = startI + lenI}
	}

	if te == 0 {return "", errors.New("no stop symbol "+ endS +" in " + source)}

	return string(source[ts:te]), nil
}
