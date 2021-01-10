package part

import (
	Ppart "github.com/qydysky/part/linuxwin"
	"os/exec"
)

type lexec struct {}

func Exec() *lexec{
	return &lexec{}
}

func (this *lexec) Run(hide bool,prog string,cmd ...string){
    Ppart.PRun(hide,prog,cmd ...)
}

func (this *lexec) Start(pro ...*exec.Cmd){
    Ppart.PStartf(pro)
}

func (this *lexec) Stop(pro ...*exec.Cmd){
    for i := range pro {
        pro[i].Process.Kill()
    }
}