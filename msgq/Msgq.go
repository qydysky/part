package part

type msgq struct {
	o chan interface{}
	men chan bool
}

type cancle int

func New() (*msgq) {
	l := new(msgq)
	(*l).o = make(chan interface{})
	(*l).men = make(chan bool)
	return l
}

func (m *msgq) Push(msg interface{}) {
	if len(m.men) == 0  {return}
	for <- m.men {
		m.o <- msg
	}
	for len(m.o) != 0 {
		<- m.o
	}
}

func (m *msgq) Pull(cancleId int) (o interface{}) {
	m.men <- true
	for {
		o = <- m.o
		if v,ok := o.(cancle);!ok || int(v) != cancleId {break}
	}
	return
}

func (m *msgq) Cancle(cancleId int) {
	m.Push(cancle(cancleId))
}