package part

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/miekg/dns"
	pe "github.com/qydysky/part/errors"
	pfile "github.com/qydysky/part/file"
	psync "github.com/qydysky/part/sync"
	us "github.com/qydysky/part/unsafe"
)

var (
	ErrDnsNoAnswer      = errors.New("ErrDnsNoAnswer")
	ErrNetworkNoSupport = errors.New("ErrNetworkNoSupport")
	ErrUdpOverflow      = errors.New("ErrUdpOverflow")
)

type netl struct {
	RV  []interface{}
	Dns dnsl
}

func Net() *netl {
	return new(netl)
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

// 桥
func connBridge(a, b net.Conn, bufSize int) {
	var wg = make(chan struct{}, 3)

	buf := make([]byte, bufSize*2)

	go func() {
		buf := buf[:bufSize]
		for {
			if n, err := a.Read(buf); err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) && !errors.Is(err, net.ErrClosed) {
					fmt.Println(err)
				}
				break
			} else if _, err = b.Write(buf[:n]); err != nil {
				if !errors.Is(err, io.EOF) {
					fmt.Println(err)
				}
				break
			}
		}
		wg <- struct{}{}
	}()

	go func() {
		buf := buf[bufSize:]
		for {
			if n, err := b.Read(buf); err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) && !errors.Is(err, net.ErrClosed) {
					fmt.Println(err)
				}
				break
			} else if _, err = a.Write(buf[:n]); err != nil {
				if !errors.Is(err, io.EOF) {
					fmt.Println(err)
				}
				break
			}
		}
		wg <- struct{}{}
	}()

	<-wg
	a.Close()
	b.Close()
}

var (
	ErrForwardAccept pe.Action = `ErrForwardAccept`
	ErrForwardDail   pe.Action = `ErrForwardDail`
)

type ForwardMsgFunc interface {
	ErrorMsg(targetaddr, listenaddr string, e error)
	WarnMsg(targetaddr, listenaddr string, e error)
	AcceptMsg(remote net.Addr, targetaddr string) (ConFinMsg func())
	ConnMsg(proxyconn, targetconn net.Conn) (ConFinMsg func())
	DenyMsg(remote net.Addr, targetaddr string)
	LisnMsg(targetaddr, listenaddr string)
	ClosMsg(targetaddr, listenaddr string)
}

func Forward(targetaddr, listenaddr string, acceptCIDRs []string, callBack ForwardMsgFunc) (closef func()) {
	closef = func() {}

	lisNet := strings.Split(listenaddr, "://")[0]
	lisAddr := strings.Split(listenaddr, "://")[1]
	tarNet := strings.Split(targetaddr, "://")[0]
	tarAddr := strings.Split(targetaddr, "://")[1]
	lisIsUdp := strings.Contains(lisNet, "udp")
	tarIsUdp := strings.Contains(tarNet, "udp")

	if (!lisIsUdp && tarIsUdp) || (lisIsUdp && !tarIsUdp) {
		callBack.ErrorMsg(targetaddr, listenaddr, ErrNetworkNoSupport)
		return
	}

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
			callBack.ErrorMsg(targetaddr, listenaddr, err)
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
			callBack.ErrorMsg(targetaddr, listenaddr, err)
			return
		}
	}

	//初始化关闭方法
	closef = func() {
		listener.Close()
		callBack.ClosMsg(targetaddr, listenaddr)
	}

	//返回监听地址
	callBack.LisnMsg(targetaddr, listenaddr)

	matchfunc := []func(ip net.IP) bool{}
	for _, cidr := range acceptCIDRs {
		if _, cidrx, err := net.ParseCIDR(cidr); err != nil {
			callBack.ErrorMsg(targetaddr, listenaddr, err)
			return
		} else {
			matchfunc = append(matchfunc, cidrx.Contains)
		}
	}

	//开始准备转发
	go func(listener net.Listener) {
		defer listener.Close()

		for {
			proxyconn, err := listener.Accept()
			if errors.Is(err, ErrUdpConnected) {
				continue
			}
			if err != nil {
				//返回Accept错误
				callBack.WarnMsg(targetaddr, listenaddr, pe.Join(ErrForwardAccept, err))
				continue
			}

			host, _, err := net.SplitHostPort(proxyconn.RemoteAddr().String())
			if err != nil {
				callBack.WarnMsg(targetaddr, listenaddr, err)
				continue
			}

			ip := net.ParseIP(host)

			var accept bool
			for i := 0; !accept && i < len(matchfunc); i++ {
				accept = accept || matchfunc[i](ip)
			}
			if !accept {
				//返回Deny
				callBack.DenyMsg(proxyconn.RemoteAddr(), targetaddr)
				proxyconn.Close()
				continue
			}
			//返回Accept
			conFin := callBack.AcceptMsg(proxyconn.RemoteAddr(), targetaddr)

			go func() {
				defer conFin()
				targetconn, err := net.Dial(tarNet, tarAddr)
				if err != nil {
					callBack.WarnMsg(targetaddr, listenaddr, pe.Join(ErrForwardDail, err))
					return
				}

				defer callBack.ConnMsg(proxyconn, targetconn)()

				// if !lisIsUdp && !tarIsUdp {
				// 	tcp2tcpConnBridge(proxyconn, targetconn, 65535)
				// } else if lisIsUdp && tarIsUdp {
				connBridge(proxyconn, targetconn, 65535)
				// } else {
				// 	connBridge(proxyconn, targetconn, 65535)
				// }
			}()
		}
	}(listener)

	return
}

var ErrUdpConnOverflow = errors.New(`ErrUdpConnOverflow`)

type udpConn struct {
	e         error
	conn      *net.UDPConn
	remoteAdd *net.UDPAddr
	ctx       context.Context
	ctxCancel context.CancelFunc
	buf       chan []byte
	closef    func() error
}

func (t *udpConn) SetBuf(b []byte) {
	tmp := make([]byte, len(b))
	copy(tmp, b)
	select {
	case t.buf <- tmp:
	default:
		t.e = ErrUdpConnOverflow
	}
}
func (t *udpConn) Read(b []byte) (n int, err error) {
	select {
	case tmp := <-t.buf:
		n = copy(b, tmp)
	case <-t.ctx.Done():
		err = os.ErrDeadlineExceeded
	}
	return
}
func (t *udpConn) Write(b []byte) (n int, err error) {
	select {
	case <-t.ctx.Done():
		err = os.ErrDeadlineExceeded
	default:
		n, err = t.conn.WriteToUDP(b, t.remoteAdd)
		if err != nil {
			t.ctx.Done()
		}
	}
	return
}
func (t *udpConn) Close() error {
	t.ctxCancel()
	return t.closef()
}
func (t *udpConn) LocalAddr() net.Addr  { return t.conn.LocalAddr() }
func (t *udpConn) RemoteAddr() net.Addr { return t.remoteAdd }
func (t *udpConn) SetDeadline(b time.Time) error {
	time.AfterFunc(time.Until(b), func() {
		t.Close()
	})
	return nil
}
func (t *udpConn) SetReadDeadline(b time.Time) error  { return t.SetDeadline(b) }
func (t *udpConn) SetWriteDeadline(b time.Time) error { return t.SetDeadline(b) }

type udpLis struct {
	udpAddr *net.UDPAddr
	c       <-chan *udpConn
	closef  func() error
}

var ErrUdpConnected error = errors.New("ErrUdpConnected")

func NewUdpListener(network, listenaddr string) (*udpLis, error) {
	udpAddr, err := net.ResolveUDPAddr(network, listenaddr)
	if err != nil {
		return nil, err
	}
	if conn, err := net.ListenUDP(network, udpAddr); err != nil {
		return nil, err
	} else {
		c := make(chan *udpConn, 10)
		lis := &udpLis{
			udpAddr: udpAddr,
			c:       c,
			closef: func() error {
				return conn.Close()
			},
		}
		go func() {
			var link psync.MapG[string, *udpConn]
			buf := make([]byte, humanize.MByte)
			for {
				n, remoteAdd, e := conn.ReadFromUDP(buf)
				if e != nil {
					c <- &udpConn{e: e}
					return
				}
				if udpc, ok := link.Load(remoteAdd.String()); ok {
					udpc.SetBuf(buf[:n])
					c <- &udpConn{e: ErrUdpConnected}
				} else {
					udpc := &udpConn{
						conn:      conn,
						remoteAdd: remoteAdd,
						closef: func() error {
							link.Delete(remoteAdd.String())
							return nil
						},
						buf: make(chan []byte, 5),
					}
					udpc.ctx, udpc.ctxCancel = context.WithTimeout(context.Background(), time.Second*30)
					udpc.SetBuf(buf[:n])
					link.Store(remoteAdd.String(), udpc)
					c <- udpc
				}
			}
		}()
		return lis, nil
	}
}
func (t *udpLis) Accept() (net.Conn, error) {
	udpc := <-t.c
	return udpc, udpc.e
}
func (t *udpLis) Close() error {
	return t.closef()
}
func (t *udpLis) Addr() net.Addr {
	return t.udpAddr
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

		this.Dns.Server = us.B2S(ip)
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
		this.Dns.Server = us.B2S(ip)
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
