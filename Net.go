package part

import (
	"bytes"
	"errors"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	pool "github.com/qydysky/part/pool"
)

type netl struct {
	RV  []interface{}
	Dns dnsl
}

func Net() *netl {
	return &netl{}
}

type dnsl struct {
	Server string
}

func (this *netl) Nslookup(target string) error {
	c := dns.Client{}
	m := dns.Msg{}

	m.SetQuestion(target+".", dns.TypeA)

	if this.Dns.Server == "" {
		if e := this.GetLocalDns(); e != nil {
			return e
		}
	}

	r, _, err := c.Exchange(&m, this.Dns.Server+":53")
	if err != nil {
		return err
	}
	if len(r.Answer) == 0 {
		return errors.New("no answer")
	}

	this.RV = append(this.RV, dns.Field(r.Answer[0], 1))
	return nil
}

func (*netl) TestDial(network, address string, Timeout int) bool {
	conn, err := net.DialTimeout(network, address, time.Duration(Timeout)*time.Second)
	if err != nil {
		Logf().E(err.Error())
		return false
	}
	conn.Close()
	return true
}

const (
	ErrorMsg = iota
	AcceptMsg
	DenyMsg
	LisnMsg
)

// when Type is ErrorMsg, Msg is set to error
// when Type is AcceptMsg, Msg is set to net.Addr
// when Type is DenyMsg, Msg is set to net.Addr
// when Type is LisnMsg, Msg is set to net.Addr
type ForwardMsg struct {
	Type int
	Msg  interface{}
}

// 桥
func connBridge(a, b net.Conn) {
	fin := make(chan bool, 1)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer func() {
			a.Close()
			b.Close()
			fin <- true
			wg.Done()
		}()

		buf := make([]byte, 20480)

		for {
			select {
			case <-fin:
				return
			default:
				n, err := a.Read(buf)

				if err != nil {
					return
				}
				b.Write(buf[:n])
			}
		}
	}()

	go func() {
		defer func() {
			a.Close()
			b.Close()
			fin <- true
			wg.Done()
		}()

		buf := make([]byte, 20480)

		for {
			select {
			case <-fin:
				return
			default:
				n, err := b.Read(buf)

				if err != nil {
					return
				}
				a.Write(buf[:n])
			}
		}
	}()

	wg.Wait()
}

// "tcp", "tcp4", "tcp6", "unix" or "unixpacket"
func Forward(targetaddr, network, listenaddr string, acceptCIDRs []string) (closef func(), msg_chan chan ForwardMsg) {
	//初始化消息通道
	msg_chan = make(chan ForwardMsg, 1000)

	//尝试监听
	listener, err := net.Listen(network, listenaddr)
	if err != nil {
		select {
		default:
		case msg_chan <- ForwardMsg{
			Type: ErrorMsg,
			Msg:  err,
		}:
		}
		return
	}

	closec := make(chan struct{})
	//初始化关闭方法
	closef = func() {
		listener.Close()
		close(closec)
	}

	//返回监听地址
	select {
	default:
	case msg_chan <- ForwardMsg{
		Type: LisnMsg,
		Msg:  listener.Addr(),
	}:
	}

	matchfunc := []func(ip net.IP) bool{}

	for _, cidr := range acceptCIDRs {
		if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
			select {
			default:
			case msg_chan <- ForwardMsg{
				Type: ErrorMsg,
				Msg:  err,
			}:
			}
			return
		} else {
			matchfunc = append(matchfunc, cidrx.Contains)
		}
	}

	//开始准备转发
	go func(listener net.Listener, targetaddr, network string, msg_chan chan ForwardMsg) {
		defer close(msg_chan)
		defer listener.Close()

		for {
			proxyconn, err := listener.Accept()
			if err != nil {
				//返回Accept错误
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: ErrorMsg,
					Msg:  err,
				}:
				}
				continue
			}

			ip := net.ParseIP(strings.Split(proxyconn.RemoteAddr().String(), ":")[0])

			var deny bool
			for i := 0; !deny && i < len(matchfunc); i++ {
				deny = deny && !matchfunc[i](ip)
			}
			if deny {
				//返回Deny
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: DenyMsg,
					Msg:  proxyconn.RemoteAddr(),
				}:
				}
				proxyconn.Close()
				continue
			}

			//返回Accept
			select {
			default:
			case msg_chan <- ForwardMsg{
				Type: AcceptMsg,
				Msg:  proxyconn.RemoteAddr(),
			}:
			}

			select {
			case <-closec:
			default:
				break
			}

			targetconn, err := net.Dial(network, targetaddr)
			if err != nil {
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: ErrorMsg,
					Msg:  err,
				}:
				}
				proxyconn.Close()
				continue
			}

			go connBridge(proxyconn, targetconn)
		}
	}(listener, targetaddr, network, msg_chan)

	return
}

func ForwardUdp(targetaddr, network, listenaddr string, acceptCIDRs []string) (closef func(), msg_chan chan ForwardMsg) {
	//初始化消息通道
	msg_chan = make(chan ForwardMsg, 1000)

	lisAddr := func(network, listenaddr string) (*net.UDPConn, error) {
		if udpAddr, err := net.ResolveUDPAddr(network, listenaddr); err != nil {
			return nil, err
		} else if conn, err := net.ListenUDP(network, udpAddr); err != nil {
			return nil, err
		} else {
			return conn, nil
		}
	}

	targetAddr, err := net.ResolveUDPAddr(network, targetaddr)
	if err != nil {
		select {
		default:
		case msg_chan <- ForwardMsg{
			Type: ErrorMsg,
			Msg:  err,
		}:
		}
		return
	}

	serConn, err := lisAddr(network, listenaddr)
	if err != nil {
		select {
		default:
		case msg_chan <- ForwardMsg{
			Type: ErrorMsg,
			Msg:  err,
		}:
		}
		return
	}

	closec := make(chan struct{})
	//初始化关闭方法
	closef = func() {
		serConn.Close()
		close(closec)
	}

	//返回监听端口
	select {
	default:
	case msg_chan <- ForwardMsg{
		Type: LisnMsg,
		Msg:  serConn.LocalAddr(),
	}:
	}

	matchfunc := []func(ip net.IP) bool{}

	for _, cidr := range acceptCIDRs {
		if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
			select {
			default:
			case msg_chan <- ForwardMsg{
				Type: ErrorMsg,
				Msg:  err,
			}:
			}
			return
		} else {
			matchfunc = append(matchfunc, cidrx.Contains)
		}
	}

	//开始准备转发
	go func(serConn *net.UDPConn, targetAddr *net.UDPAddr, msg_chan chan ForwardMsg) {
		defer close(msg_chan)
		defer serConn.Close()

		type UDPConn struct {
			p      *net.UDPConn
			pooled bool
		}

		var (
			connMap = make(map[string]*UDPConn)
			udpPool = pool.New[UDPConn](pool.PoolFunc[UDPConn]{
				New: func() *UDPConn {
					conn, _ := net.ListenUDP(network, nil)
					return &UDPConn{p: conn}
				},
				InUse: func(u *UDPConn) bool {
					return !u.pooled
				},
				Reuse: func(u *UDPConn) *UDPConn {
					u.pooled = false
					return u
				},
				Pool: func(u *UDPConn) *UDPConn {
					u.pooled = true
					return u
				},
			}, 100)
		)
		genConn := func(cliAddr *net.UDPAddr) *UDPConn {
			if conn, ok := connMap[cliAddr.String()]; ok {
				return conn
			} else {
				conn = udpPool.Get()
				connMap[cliAddr.String()] = conn
				return conn
			}
		}

		buf := make([]byte, 20480)
		for {
			n, cliAddr, err := serConn.ReadFromUDP(buf)
			if err != nil {
				//返回Accept错误
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: ErrorMsg,
					Msg:  err,
				}:
				}
				continue
			}

			var accpet bool
			for i := 0; i < len(matchfunc); i++ {
				accpet = accpet || matchfunc[i](cliAddr.IP)
			}
			if !accpet {
				//返回Deny
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: DenyMsg,
					Msg:  net.Addr(cliAddr),
				}:
				}
				continue
			}

			//返回Accept
			select {
			default:
			case msg_chan <- ForwardMsg{
				Type: AcceptMsg,
				Msg:  net.Addr(cliAddr),
			}:
			}

			select {
			case <-closec:
			default:
				break
			}

			targetConn := genConn(cliAddr)
			if _, err := targetConn.p.WriteToUDP(buf[:n], targetAddr); err != nil {
				//返回Accept错误
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: ErrorMsg,
					Msg:  err,
				}:
				}
			} else {
				go func() {
					defer udpPool.Put(targetConn)

					buf := make([]byte, 20480)
					for {
						targetConn.p.SetDeadline(time.Now().Add(time.Second * 20))
						n, _, e := targetConn.p.ReadFromUDP(buf)
						if e != nil {
							return
						}
						if n, e = targetConn.p.WriteToUDP(buf[:n], cliAddr); err != nil {
							return
						}
					}
				}()
			}
		}
	}(serConn, targetAddr, msg_chan)

	return
}

func (this *netl) GetLocalDns() error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("nslookup", "127.0.0.1")
		output, _ := cmd.CombinedOutput()
		var ip []byte
		loc_ip := bytes.Index(output, []byte("Address:")) + 8

		for brk := 1; brk >= 0; {
			tmp := bytes.IndexAny(output[loc_ip:], "1234567890.")
			if tmp == 0 {
				ip = append(ip, output[loc_ip])
				loc_ip = loc_ip + 1
			} else {
				brk = brk - 1
				loc_ip = loc_ip + tmp
			}
		}

		this.Dns.Server = string(ip)
		return nil
	} else if Checkfile().IsExist("/etc/resolv.conf") {
		cmd := exec.Command("cat", "/etc/resolv.conf")
		output, _ := cmd.CombinedOutput()
		var ip []byte
		loc_ip := bytes.Index(output, []byte("nameserver")) + 10
		for brk := 1; brk >= 0; {
			tmp := bytes.IndexAny(output[loc_ip:], "1234567890.")
			if tmp == 0 {
				ip = append(ip, output[loc_ip])
				loc_ip = loc_ip + 1
			} else {
				brk = brk - 1
				loc_ip = loc_ip + tmp
			}
		}
		this.Dns.Server = string(ip)
		return nil
	}
	Logf().E("[err]Dns: system: ", runtime.GOOS)
	Logf().E("[err]Dns: none")

	return errors.New("1")
}

func MasterDomain(url_s string) (string, error) {
	if u, e := url.Parse(url_s); e != nil {
		return "", e
	} else {
		host := u.Hostname()
		list := strings.Split(host, ".")
		if len(list) < 2 {
			return "", errors.New("invalid domain:" + host)
		}
		return strings.Join(list[len(list)-2:], "."), nil
	}
	return "", nil
}
