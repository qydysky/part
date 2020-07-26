package part

import (
	"sync"
    "path/filepath"
	"os"
	"runtime"
	"time"
	"net"
	"strconv"
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

func (this *sys) GetFreeProt() int {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
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
