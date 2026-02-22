package part

import (
	"os/exec"

	Ppart "github.com/qydysky/part/linuxwin"
)

type lexec struct{}

func Exec() *lexec {
	return new(lexec)
}

func (this *lexec) Run(hide bool, prog string, cmd ...string) {
	Ppart.PRun(hide, prog, cmd...)
}

func (this *lexec) Start(pro ...*exec.Cmd) {
	Ppart.PStartf(pro)
}

func (this *lexec) Stop(pro ...*exec.Cmd) {
	for i := range pro {
		if pro[i] == nil {
			continue
		}
		pro[i].Process.Kill()
	}
}
