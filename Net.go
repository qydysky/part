package part

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	pfile "github.com/qydysky/part/file"
)

var (
	ErrDnsNoAnswer      = errors.New("ErrDnsNoAnswer")
	ErrNetworkNoSupport = errors.New("ErrNetworkNoSupport")
	ErrUdpOverflow      = errors.New("ErrUdpOverflow") // default:1500 set higher pkgSize at NewUdpListener
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
		return ErrDnsNoAnswer
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
	WarnMsg
	AcceptMsg
	DenyMsg
	LisnMsg
	ClosMsg
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
func connBridge(a, b net.Conn, bufSize int) {
	fmt.Println(b.LocalAddr())
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer func() {
			fmt.Println("close r", b.LocalAddr())
			wg.Done()
		}()

		buf := make([]byte, bufSize)

		for {
			if n, err := a.Read(buf); err != nil {
				a.Close()
				return
			} else if _, err = b.Write(buf[:n]); err != nil {
				fmt.Println(err)
				return
			}
		}
	}()

	go func() {
		defer func() {
			fmt.Println("close w", b.LocalAddr())
			wg.Done()
		}()

		buf := make([]byte, bufSize)

		for {
			if n, err := b.Read(buf); err != nil {
				b.Close()
				return
			} else if _, err = a.Write(buf[:n]); err != nil {
				fmt.Println(err)
				return
			}
		}
	}()

	wg.Wait()
	fmt.Println("close", b.LocalAddr())
}

func Forward(targetaddr, listenaddr string, acceptCIDRs []string) (closef func(), msg_chan chan ForwardMsg) {
	msg_chan = make(chan ForwardMsg, 1000)
	closef = func() {}

	lisNet := strings.Split(listenaddr, "://")[0]
	lisAddr := strings.Split(listenaddr, "://")[1]
	tarNet := strings.Split(targetaddr, "://")[0]
	tarAddr := strings.Split(targetaddr, "://")[1]

	//尝试监听
	var listener net.Listener
	{
		var err error
		switch lisNet {
		case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
			listener, err = net.Listen(lisNet, lisAddr)
		case "udp", "udp4", "udp6":
			listener, err = NewUdpListener(lisNet, lisAddr)
		default:
			err = ErrNetworkNoSupport
		}
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
	}
	{
		var err error
		switch lisNet {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "ip", "ip4", "ip6", "unix", "unixgram", "unixpacket":
		default:
			err = ErrNetworkNoSupport
		}
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
	go func(listener net.Listener, msg_chan chan ForwardMsg) {
		defer close(msg_chan)
		defer listener.Close()

		for {
			proxyconn, err := listener.Accept()
			if err != nil {
				//返回Accept错误
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: WarnMsg,
					Msg:  err,
				}:
				}
				continue
			}

			host, _, err := net.SplitHostPort(proxyconn.RemoteAddr().String())
			if err != nil {
				select {
				default:
				case msg_chan <- ForwardMsg{
					Type: WarnMsg,
					Msg:  err,
				}:
				}
				continue
			}

			ip := net.ParseIP(host)

			var accept bool
			for i := 0; !accept && i < len(matchfunc); i++ {
				accept = accept || matchfunc[i](ip)
			}
			if !accept {
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

			go func() {
				targetconn, err := net.Dial(tarNet, tarAddr)
				if err != nil {
					select {
					default:
					case msg_chan <- ForwardMsg{
						Type: WarnMsg,
						Msg:  err,
					}:
					}
					return
				}

				connBridge(proxyconn, targetconn, 20480)

				switch lisNet {
				case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
				case "udp", "udp4", "udp6":
					time.Sleep(time.Second * 10)
				default:
				}
			}()
		}
	}(listener, msg_chan)

	return
}

type udpConn struct {
	conn      *net.UDPConn
	remoteAdd *net.UDPAddr
	reader    io.Reader
}

func NewUdpConn(per []byte, remoteAdd *net.UDPAddr, conn *net.UDPConn) *udpConn {
	return &udpConn{
		conn:      conn,
		remoteAdd: remoteAdd,
		reader:    bytes.NewReader(per),
	}
}
func (t udpConn) Read(b []byte) (n int, err error) {
	return t.reader.Read(b)
}
func (t udpConn) Write(b []byte) (n int, err error) {
	return t.conn.WriteToUDP(b, t.remoteAdd)
}
func (t udpConn) Close() error                       { return nil }
func (t udpConn) LocalAddr() net.Addr                { return t.conn.LocalAddr() }
func (t udpConn) RemoteAddr() net.Addr               { return t.remoteAdd }
func (t udpConn) SetDeadline(b time.Time) error      { return t.conn.SetDeadline(b) }
func (t udpConn) SetReadDeadline(b time.Time) error  { return t.conn.SetReadDeadline(b) }
func (t udpConn) SetWriteDeadline(b time.Time) error { return t.conn.SetWriteDeadline(b) }

type udpLis struct {
	addr    net.Addr
	conn    *net.UDPConn
	pkgSize int
}

func NewUdpListener(network, listenaddr string, pkgSize ...int) (*udpLis, error) {
	udpAddr, err := net.ResolveUDPAddr(network, listenaddr)
	if err != nil {
		return nil, err
	}
	pkgSize = append(pkgSize, 1500)
	if conn, err := net.ListenUDP(network, udpAddr); err != nil {
		return nil, err
	} else {
		return &udpLis{
			addr:    udpAddr,
			conn:    conn,
			pkgSize: pkgSize[0],
		}, nil
	}
}
func (t udpLis) Accept() (net.Conn, error) {
	buf := make([]byte, t.pkgSize)
	if n, remoteAdd, e := t.conn.ReadFromUDP(buf); e != nil {
		return nil, e
	} else if n == t.pkgSize {
		return nil, ErrUdpOverflow
	} else {
		return NewUdpConn(buf[:n], remoteAdd, t.conn), nil
	}
}
func (t udpLis) Close() error {
	return t.conn.Close()
}
func (t udpLis) Addr() net.Addr {
	return t.addr
}

// type ConnPool struct {
// 	pool *pool.Buf[ConnPoolItem]
// }

// type ConnPoolItem struct {
// 	p      *net.Conn
// 	pooled bool
// }

// func NewConnPool(size int, genConn func() *net.Conn) *ConnPool {
// 	return &ConnPool{
// 		pool: pool.New(pool.PoolFunc[ConnPoolItem]{
// 			New: func() *ConnPoolItem {
// 				return &ConnPoolItem{p: genConn()}
// 			},
// 			InUse: func(u *ConnPoolItem) bool {
// 				return !u.pooled
// 			},
// 			Reuse: func(u *ConnPoolItem) *ConnPoolItem {
// 				u.pooled = false
// 				return u
// 			},
// 			Pool: func(u *ConnPoolItem) *ConnPoolItem {
// 				u.pooled = true
// 				return u
// 			},
// 		}, size),
// 	}
// }

// func (t *ConnPool) Get(id string) (i net.Conn, putBack func()) {
// 	conn := t.pool.Get()
// 	return *conn.p, func() {
// 		t.pool.Put(conn)
// 	}
// }

// func ForwardUdp(targetaddr, network, listenaddr string, acceptCIDRs []string) (closef func(), msg_chan chan ForwardMsg) {
// 	//初始化消息通道
// 	msg_chan = make(chan ForwardMsg, 1000)

// 	targetAddr, err := net.ResolveUDPAddr(network, targetaddr)
// 	if err != nil {
// 		select {
// 		default:
// 		case msg_chan <- ForwardMsg{
// 			Type: ErrorMsg,
// 			Msg:  err,
// 		}:
// 		}
// 		return
// 	}

// 	serConn, err := udpListener(network, listenaddr)
// 	if err != nil {
// 		select {
// 		default:
// 		case msg_chan <- ForwardMsg{
// 			Type: ErrorMsg,
// 			Msg:  err,
// 		}:
// 		}
// 		return
// 	}

// 	closec := make(chan struct{})
// 	//初始化关闭方法
// 	closef = func() {
// 		serConn.Close()
// 		close(closec)
// 	}

// 	//返回监听端口
// 	select {
// 	default:
// 	case msg_chan <- ForwardMsg{
// 		Type: LisnMsg,
// 		Msg:  serConn.LocalAddr(),
// 	}:
// 	}

// 	matchfunc := []func(ip net.IP) bool{}

// 	for _, cidr := range acceptCIDRs {
// 		if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
// 			select {
// 			default:
// 			case msg_chan <- ForwardMsg{
// 				Type: ErrorMsg,
// 				Msg:  err,
// 			}:
// 			}
// 			return
// 		} else {
// 			matchfunc = append(matchfunc, cidrx.Contains)
// 		}
// 	}

// 	//开始准备转发
// 	go func(serConn net.Conn, targetAddr *net.UDPAddr, msg_chan chan ForwardMsg) {
// 		defer close(msg_chan)
// 		defer serConn.Close()

// 		type UDPConn struct {
// 			p      *net.UDPConn
// 			pooled bool
// 		}

// 		var (
// 			connMap = make(map[string]*UDPConn)
// 			udpPool = pool.New[UDPConn](pool.PoolFunc[UDPConn]{
// 				New: func() *UDPConn {
// 					conn, _ := net.ListenUDP(network, nil)
// 					return &UDPConn{p: conn}
// 				},
// 				InUse: func(u *UDPConn) bool {
// 					return !u.pooled
// 				},
// 				Reuse: func(u *UDPConn) *UDPConn {
// 					u.pooled = false
// 					return u
// 				},
// 				Pool: func(u *UDPConn) *UDPConn {
// 					u.pooled = true
// 					return u
// 				},
// 			}, 100)
// 		)
// 		genConn := func(cliAddr *net.UDPAddr) *UDPConn {
// 			if conn, ok := connMap[cliAddr.String()]; ok {
// 				return conn
// 			} else {
// 				conn = udpPool.Get()
// 				connMap[cliAddr.String()] = conn
// 				return conn
// 			}
// 		}

// 		buf := make([]byte, 20480)
// 		for {
// 			n, err := serConn.Read(buf)
// 			if err != nil {
// 				//返回Accept错误
// 				select {
// 				default:
// 				case msg_chan <- ForwardMsg{
// 					Type: ErrorMsg,
// 					Msg:  err,
// 				}:
// 				}
// 				continue
// 			}

// 			ip := net.ParseIP(strings.Split(serConn.RemoteAddr().String(), ":")[0])
// 			var deny bool
// 			for i := 0; !deny && i < len(matchfunc); i++ {
// 				deny = deny && !matchfunc[i](ip)
// 			}
// 			if deny {
// 				//返回Deny
// 				select {
// 				default:
// 				case msg_chan <- ForwardMsg{
// 					Type: DenyMsg,
// 					Msg:  serConn.RemoteAddr(),
// 				}:
// 				}
// 				serConn.Close()
// 				continue
// 			}

// 			//返回Accept
// 			select {
// 			default:
// 			case msg_chan <- ForwardMsg{
// 				Type: AcceptMsg,
// 				Msg:  serConn.RemoteAddr(),
// 			}:
// 			}

// 			select {
// 			case <-closec:
// 			default:
// 				break
// 			}

// 			targetConn := genConn(cliAddr)
// 			if _, err := targetConn.p.WriteToUDP(buf[:n], targetAddr); err != nil {
// 				//返回Accept错误
// 				select {
// 				default:
// 				case msg_chan <- ForwardMsg{
// 					Type: ErrorMsg,
// 					Msg:  err,
// 				}:
// 				}
// 			} else {
// 				go func() {
// 					defer udpPool.Put(targetConn)

// 					buf := make([]byte, 20480)
// 					for {
// 						targetConn.p.SetDeadline(time.Now().Add(time.Second * 20))
// 						n, _, e := targetConn.p.ReadFromUDP(buf)
// 						if e != nil {
// 							return
// 						}
// 						if n, e = targetConn.p.WriteToUDP(buf[:n], cliAddr); err != nil {
// 							return
// 						}
// 					}
// 				}()
// 			}
// 		}
// 	}(serConn, targetAddr, msg_chan)

// 	return
// }

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
	} else if pfile.New("/etc/resolv.conf", 0, true).IsExist() {
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
}
