package part

type msgq struct {
	d interface{}
	i chan struct{}
	o chan struct{}
}

func New() (*msgq) {
	l := new(msgq)
	(*l).i = make(chan struct{})
	(*l).o = make(chan struct{})
	close((*l).i)
	return l
}

func (m *msgq) Push(msg interface{}) {
	m.i = make(chan struct{})
	m.d = msg
	close(m.o)
	m.o = make(chan struct{})
	close(m.i)
}

func (m *msgq) Pull() (o interface{}) {
	<- m.i
	<- m.o
	return m.d
}
