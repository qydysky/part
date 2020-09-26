package part

type signal struct{
	v chan struct{}
}

func Signal() (o *signal) {
	return &signal{v:make(chan struct{})}
}

func (i *signal) Wait() {
	<-i.v
}

func (i *signal) Done() {
	if i.Islive() {close(i.v)}
}

func (i *signal) Islive() (islive bool) {
	select {
	case <-i.v:;
	default:
		if i.v == nil {break}
		islive = true
	}
	return
}