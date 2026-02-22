package part

import (
	"errors"
	"fmt"
	"iter"
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

var EOL = Ppart.EOF

type sys struct {
	RV []interface{}
	sync.Mutex
}

func Sys() *sys {
	return new(sys)
}

func (*sys) Type(s ...interface{}) string {
	if len(s) == 0 {
		return "nil"
	}
	switch t := s[0].(type) {
	default:
		return fmt.Sprintf("%T", t)
	}
}

func (t *sys) Cdir() string {
	t.Lock()
	defer t.Unlock()

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
		fmt.Println(cdir, "LastIndex", s, "-1")
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

func (t *sys) Timeoutf(Timeout int) {
	t.Lock()
	defer t.Unlock()

	time.Sleep(time.Duration(Timeout) * time.Second)
}

func (t *sys) MTimeoutf(Millisecond int) {
	t.Lock()
	defer t.Unlock()

	time.Sleep(time.Duration(Millisecond) * time.Millisecond)
}

func (t *sys) GetSys(sys string) bool {
	t.RV = append(t.RV, runtime.GOOS)
	return runtime.GOOS == sys
}

func (t *sys) GetTime() string {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	return now[len(now)-4:]
}

func (t *sys) GetMTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
func (t *sys) GetSTime() int64 {
	return time.Now().Unix()
}

func (t *sys) GetFreePort() int {
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

	dir, err := os.MkdirTemp(pdir, "")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	return dir
}

func (t *sys) GetTmpFile(pdir string) string {
	defer t.Unlock()
	t.Lock()

	tmpfile, err := os.MkdirTemp(pdir, "*.tmp")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	name := tmpfile
	return name
}

// Deprecated: use GetIpByCidr
func GetIntranetIp(cidr string) (ips []string) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
		fmt.Println("[part]no loacl ip")
		return []string{"127.0.0.1"}
	}

	var (
		cidrN *net.IPNet
	)
	if cidr == `` {
		cidr = `0.0.0.0/0`
	}
	_, cidrN, err = net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println("[part]cidr incorrect")
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						if cidrN != nil && cidrN.Contains(ipnet.IP) {
							ips = append(ips, ipnet.IP.String())
						}
					}
				}
			}
		}
	}

	if len(ips) != 0 {
		return
	}

	fmt.Println("[part]no loacl ip")
	return []string{"127.0.0.1"}
}

func GetIpByCidr(cidr ...string) (seq iter.Seq[net.IP]) {
	return func(yield func(net.IP) bool) {
		netInterfaces, err := net.Interfaces()
		if err != nil {
			return
		}

		var cidrN *net.IPNet
		if len(cidr) == 0 {
			cidr = append(cidr, `0.0.0.0/0`)
			cidr = append(cidr, `::/0`)
		}

		for i := 0; i < len(cidr); i++ {
			_, cidrN, err = net.ParseCIDR(cidr[i])
			if err != nil {
				return
			}

			for i := 0; i < len(netInterfaces); i++ {
				if (netInterfaces[i].Flags & net.FlagUp) != 0 {
					addrs, _ := netInterfaces[i].Addrs()

					for _, address := range addrs {
						if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsUnspecified() {
							if cidrN != nil && cidrN.Contains(ipnet.IP) {
								if !yield(ipnet.IP) {
									return
								}
							}
						}
					}
				}
			}
		}
	}
}

func (t *sys) CheckProgram(pros ...string) []int {
	return Ppart.PCheck(pros)
}

func (t *sys) SetProxy(s, pac string) error {
	return Ppart.PProxy(s, pac)
}

func (t *sys) GetCpuPercent() (float64, error) {
	if a, e := gopsutilLoad.Avg(); e == nil {
		if i, e := gopsutilCpu.Counts(true); e == nil {
			return (*a).Load1 / float64(i), nil
		} else {
			fmt.Println(e.Error())
		}
	} else {
		fmt.Println(e.Error())
	}
	return 0.0, errors.New("cant get CpuPercent")
}

func (t *sys) PreventSleep() (stop *signal.Signal) {
	return Ppart.PreventSleep()
}
