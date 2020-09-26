package part

type Signal struct{
	v chan struct{}
}

func Signal_Init() (o *Signal) {
	return &Signal{v:make(chan struct{})}
}

func (i *Signal) Wait() {
	<-i.v
}

func (i *Signal) Done() {
	if i.Islive() {close(i.v)}
}

func (i *Signal) Islive() (islive bool) {
	select {
	case <-i.v:;
	default:
		if i.v == nil {break}
		islive = true
	}
	return
}