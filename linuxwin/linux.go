//go:build linux
// +build linux

package Ppart

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	signal "github.com/qydysky/part/signal"
)

func PCheck(pros []string) []int {
	res := []int{}
	_pros := [][]byte{}

	for _, v := range pros {
		if v == "" {
			return res
		}
		_pros = append(_pros, []byte(v))
		res = append(res, 0)
	}

	for j, i := range _pros {
		cmd := exec.Command("pgrep", "-c", string(i))
		output, _ := cmd.Output()
		outputt := strings.Replace(string(output), "\n", "", -1)
		res[j], _ = strconv.Atoi(outputt)
	}
	return res
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
	if s == "off" {
		if e := PRun(true, "gsettings", "set", "org.gnome.system.proxy", "mode", "none"); e != nil {
			return e
		}
		if e := PRun(true, "kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0"); e != nil {
			return e
		}
	} else {
		if e := PRun(true, "gsettings", "set", "org.gnome.system.proxy", "autoconfig-url", pacUrl); e != nil {
			return e
		}
		if e := PRun(true, "gsettings", "set", "org.gnome.system.proxy", "mode", "auto"); e != nil {
			return e
		}
		if e := PRun(true, "kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "2"); e != nil {
			return e
		}
		if e := PRun(true, "kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "Proxy Config Script", pacUrl); e != nil {
			return e
		}
	}
	return nil
}

func FileMove(src, trg string) error {
	return os.Rename(src, trg)
}

func PreventSleep() (stop *signal.Signal) {
	return nil
}
