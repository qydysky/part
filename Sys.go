package part

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	Ppart "github.com/qydysky/part/linuxwin"
	signal "github.com/qydysky/part/signal"
	gopsutilCpu "github.com/shirou/gopsutil/cpu"
	gopsutilLoad "github.com/shirou/gopsutil/load"
)

type sys struct {
	RV []interface{}
	sync.Mutex
}

func Sys() *sys {
	return &sys{}
}

func (*sys) Type(s ...interface{}) string {
	if len(s) == 0 {
		return "nil"
	}
	switch t := s[0].(type) {
	default:
		return fmt.Sprintf("%T", t)
	}
	return ""
}

func (this *sys) Cdir() string {
	this.Lock()
	defer this.Unlock()

	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	return exPath
}

func (t *sys) Pdir(cdir string) string {
	var s string = "/"
	if t.GetSys("windows") {
		s = "\\"
	}
	if p := strings.LastIndex(cdir, s); p == -1 {
		Logf().E(cdir, "LastIndex", s, "-1")
	} else {
		return cdir[:p]
	}
	return cdir
}

func GetRV(i *[]interface{}, num int) []interface{} {
	p := (*i)[:num]
	(*i) = append((*i)[num:])
	return p
}

func (this *sys) Timeoutf(Timeout int) {
	this.Lock()
	defer this.Unlock()

	time.Sleep(time.Duration(Timeout) * time.Second)
}

func (this *sys) MTimeoutf(Millisecond int) {
	this.Lock()
	defer this.Unlock()

	time.Sleep(time.Duration(Millisecond) * time.Millisecond)
}

func (this *sys) GetSys(sys string) bool {
	this.RV = append(this.RV, runtime.GOOS)
	return runtime.GOOS == sys
}

func (this *sys) GetTime() string {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	return now[len(now)-4:]
}

func (this *sys) GetMTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
func (this *sys) GetSTime() int64 {
	return time.Now().Unix()
}

func (this *sys) GetFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func (t *sys) GetTmpDir(pdir string) string {
	defer t.Unlock()
	t.Lock()

	dir, err := ioutil.TempDir(pdir, "")
	if err != nil {
		Logf().E(err.Error())
		return ""
	}

	return dir
}

func (t *sys) GetTmpFile(pdir string) string {
	defer t.Unlock()
	t.Lock()

	tmpfile, err := ioutil.TempFile(pdir, "*.tmp")
	if err != nil {
		Logf().E(err.Error())
		return ""
	}
	name := tmpfile.Name()
	tmpfile.Close()
	return name
}

func (this *sys) GetIntranetIp() string {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		Logf().E("net.Interfaces failed, err:", err.Error())
		Logf().E("[part]no loacl ip")
		return "127.0.0.1"
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	Logf().E("[part]no loacl ip")
	return "127.0.0.1"
}

func (this *sys) CheckProgram(pros ...string) []int {
	return Ppart.PCheck(pros)
}

func (this *sys) SetProxy(s, pac string) error {
	return Ppart.PProxy(s, pac)
}

func (this *sys) GetCpuPercent() (float64, error) {
	if a, e := gopsutilLoad.Avg(); e == nil {
		if i, e := gopsutilCpu.Counts(true); e == nil {
			return (*a).Load1 / float64(i), nil
		} else {
			Logf().E(e.Error())
		}
	} else {
		Logf().E(e.Error())
	}
	return 0.0, errors.New("cant get CpuPercent")
}

func (this *sys) PreventSleep() (stop *signal.Signal) {
	return Ppart.PreventSleep()
}
