package part

type Signal struct{
	Chan chan struct{}
}

func Init() (*Signal) {
	return &Signal{Chan:make(chan struct{})}
}

func (i *Signal) Wait() {
	if i.Islive() {<-i.Chan}
}

func (i *Signal) Done() {
	if i.Islive() {close(i.Chan)}
}

func (i *Signal) Islive() (islive bool) {
	if i == nil {return}
	select {
	case <-i.Chan:;//close
	default://still alive
		if i.Chan == nil {break}//not make yet
		islive = true//has made
	}
	return
}