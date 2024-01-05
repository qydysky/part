//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package Ppart

import (
	"os"
	"os/exec"
	"path/filepath"

	signal "github.com/qydysky/part/signal"
)

func PCheck(pros []string) []int {
	return []int{}
}

func PStartf(pro []*exec.Cmd) {
	for i := range pro {
		pro[i].Start()
	}
}

func PRun(hide bool, prog string, cmd ...string) error {
	p := exec.Command(prog, cmd...)
	if hide {
	}
	e := p.Run()
	return e
}

func Cdir() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	return exPath
}

func PProxy(s, pacUrl string) error {
	return nil
}

func FileMove(src, trg string) error {
	return os.Rename(src, trg)
}

func PreventSleep() (stop *signal.Signal) {
	return nil
}
