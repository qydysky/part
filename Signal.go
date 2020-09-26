package part

type Signal struct{
	v chan struct{}
}

func (i *Signal) Signal_Init() (o *Signal) {
	o = i
	o.v = make(chan struct{})
	return
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