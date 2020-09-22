package part

import (
	"strings"
	"errors"
)

type get struct {
	body []byte
	
	RS string
	Err error
}

func Get(r Rval) (o *get){
	o = new(get)

	if r.Url == "" {o.Err = errors.New("url == nil");return}

	R := Req()
	o.Err = R.Reqf(r)
	(*o).body = R.Respon

	return
}

func (i *get) S(stratS,endS string, startI,lenI int) (o *get) {
	o = i

	if stratS == "" && startI == 0 {o.Err = errors.New("no symbol to start");return}
	if endS == "" && lenI == 0 {o.Err = errors.New("no symbol to stop");return}
	s := string(o.body)

	var ts,te int

	if stratS != "" {
		if tmp := strings.Index(s, stratS);tmp != -1{ts = tmp + len(stratS)}
	} else if startI != 0 {
		if startI < len(s){ts = startI}
	}

	if ts == 0 {o.Err = errors.New("no start symbol in " + s);return}

	if endS != "" {
		if tmp := strings.Index(s[ts:], endS);tmp != -1{te = ts + tmp}
	} else if lenI != 0 {
		if startI + lenI < len(s){te = startI + lenI}
	}

	if te == 0 {o.Err = errors.New("no stop symbol in " + s);return}

	o.RS = s[ts:te]
	o.Err = nil
	return
}
