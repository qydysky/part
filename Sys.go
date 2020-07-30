package part

import (
	"sync"
    "path/filepath"
	"os"
	"runtime"
	"time"
	"net"
	"strconv"
	"io/ioutil"

	Ppart "github.com/qydysky/part/linuxwin"
)

type sys struct {sync.Mutex}

func Sys () *sys {
	return &sys{}
}

func (this *sys) Cdir()string{
	this.Lock()
	defer this.Unlock()

    dir, _ := os.Executable()
    exPath := filepath.Dir(dir)
    return exPath
}

func (this *sys) Timeoutf(Timeout int) {
	this.Lock()
	defer this.Unlock()
	
    time.Sleep(time.Duration(Timeout)*time.Second)
}

func (this *sys) GetSys(sys string)bool{
	this.Lock()
	defer this.Unlock()

    return runtime.GOOS==sys
}

func (this *sys) GetTime() string {
	now := strconv.FormatInt(time.Now().Unix(),10)
	return now[len(now)-4:]
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
	if err != nil {Logf().E(err.Error());return ""}

	return dir
}

func (t *sys) GetTmpFile(pdir string) string {
	defer t.Unlock()
	t.Lock()

	tmpfile, err := ioutil.TempFile(pdir, "*.tmp")
	if err != nil {Logf().E(err.Error());return ""}
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
    return Ppart.PCheck(pros);
}

func (this *sys) SetProxy(s,pac string) error {
    return Ppart.PProxy(s,pac);
}