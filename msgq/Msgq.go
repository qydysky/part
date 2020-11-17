package part

type msgq struct {
	o chan interface{}
	men chan bool
}

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

func (m *msgq) Pull() (o interface{}) {
	m.men <- true
	return <- m.o
}
