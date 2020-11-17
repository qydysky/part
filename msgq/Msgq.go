package part

type msgq struct {
	o chan interface{}
	men chan bool
}

func New() (*msgq) {
	l := new(msgq)
	(*l).o = make(chan interface{},1e9)
	(*l).men = make(chan bool,1e9)
	return l
}

func (m *msgq) Push(msg interface{}) {
	num := len(m.men)
	for len(m.men) != 0 {
		<- m.men
	}
	for num > 0 {
		m.o <- msg
		num -= 1
	}
	for len(m.o) != 0 {
		<- m.o
	}
}

func (m *msgq) Pull() (o interface{}) {
	m.men <- true
	return <- m.o
}
