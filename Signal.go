package part

type Signal struct{
	Chan chan struct{}
}

func (i *Signal) Init() (o *Signal) {
	o = i
	if !i.Islive() {o.Chan = make(chan struct{})}
	return
}

func (i *Signal) Wait() {
	if i.Islive() {<-i.Chan}
}

func (i *Signal) Done() {
	if i.Islive() {close(i.Chan)}
}

func (i *Signal) Islive() (islive bool) {
	select {
	case <-i.Chan:;
	default:
		if i.Chan == nil {break}
		islive = true
	}
	return
}