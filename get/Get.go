package part

import (
	"strings"
	"errors"
	p "github.com/qydysky/part"
)

type get struct {
	body []byte
	
	RS string
	Err error
}

func Get(r p.Rval) (o *get){
	o = new(get)

	if r.Url == "" {o.Err = errors.New("url == nil");return}

	R := p.Req()
	o.Err = R.Reqf(r)
	(*o).body = R.Respon

	return
}

func (i *get) S(stratS,endS string, startI,lenI int) (o *get) {
	o = i
	o.RS,o.Err = SS(string(o.body), stratS, endS, startI, lenI)
	return
}

func SS(source,stratS,endS string, startI,lenI int) (string,error) {
	if stratS == "" && startI == 0 {return "", errors.New("no symbol to start")}
	if endS == "" && lenI == 0 {return "", errors.New("no symbol to stop")}
	s := source

	var ts,te int

	if stratS != "" {
		if tmp := strings.Index(s, stratS);tmp != -1{ts = tmp + len(stratS)}
	} else if startI != 0 {
		if startI < len(s){ts = startI}
	}

	if ts == 0 {return "", errors.New("no start symbol in " + s)}

	if endS != "" {
		if tmp := strings.Index(s[ts:], endS);tmp != -1{te = ts + tmp}
	} else if lenI != 0 {
		if startI + lenI < len(s){te = startI + lenI}
	}

	if te == 0 {return "", errors.New("no stop symbol in " + s)}

	return s[ts:te], nil
}
