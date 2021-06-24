package part

import (
    "net"
    "sync"
    "strings"
    "errors"
	"bytes"
    "net/url"
	"os/exec"
    "runtime"
    "time"
	"github.com/miekg/dns"
)

type netl struct{
    RV []interface{}
    Dns dnsl
}

func Net() *netl{
	return &netl{}
}

type dnsl struct{
    Server string
}

func (this *netl) Nslookup(target string) error {
	c := dns.Client{}
    m := dns.Msg{}
    
    m.SetQuestion(target+".", dns.TypeA)
    
    if this.Dns.Server == "" {
        if e := this.GetLocalDns(); e != nil {return e}
    }

	r, _, err := c.Exchange(&m, this.Dns.Server+":53")
	if err != nil {
		return err
    }
    if len(r.Answer) == 0 {
		return errors.New("no answer")
    }
    
    this.RV = append(this.RV, dns.Field(r.Answer[0],1))
    return nil
}

func (*netl) TestDial(network,address string, Timeout int) bool {
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
    PortMsg
)

//when Type is ErrorMsg, Msg is set to error
//when Type is AcceptMsg, Msg is set to nil
//when Type is PortMsg, Msg is set to Listen Port(int)
type ForwardMsg struct {
    Type int
    Msg interface{}
}

func Forward(targetaddr,targetnetwork *string, listenaddr string) (closef func(),msg_chan chan ForwardMsg) {
    //初始化消息通道
    msg_chan = make(chan ForwardMsg, 1000)

    //尝试监听
    listener, err := net.Listen("tcp", listenaddr + ":0")
    if err != nil {
        select{
        default:;
        case msg_chan <- ForwardMsg{
            Type: ErrorMsg,
            Msg: err,
        }:;
        }
        return
    }

    //初始化关闭方法
    closef = func(){ 
        *targetaddr, *targetnetwork = "", ""
        listener.Close()
    }
    
    //返回监听端口
    select{
    default:;
    case msg_chan <- ForwardMsg{
        Type: PortMsg,
        Msg: listener.Addr().(*net.TCPAddr).Port,
    }:;
    }

    //开始准备转发
    go func(listener net.Listener, targetaddr,targetnetwork *string, msg_chan chan ForwardMsg){
        defer close(msg_chan)
        defer listener.Close()

        //tcp 桥
        tcpBridge2 := func (a, b net.Conn) {
        
            fin:=make(chan bool,1)
            var wg sync.WaitGroup
        
            wg.Add(2)
            go func(){
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
                        return;
                    default:
                        n, err := a.Read(buf)
                
                        if err != nil {return}
                        b.Write(buf[:n])
                    }
                }
            }()
            
            go func(){
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
                        return;
                    default:
                        n, err := b.Read(buf)
                
                        if err != nil {return}
                        a.Write(buf[:n])
                    }
                }
            }()
        
            wg.Wait()
        }

        for {
            proxyconn, err := listener.Accept()
            if err != nil {
                //返回Accept错误
                select{
                default:;
                case msg_chan <- ForwardMsg{
                    Type: ErrorMsg,
                    Msg: err,
                }:;
                }
                continue
            }
            
            //返回Accept
            select{
            default:;
            case msg_chan <- ForwardMsg{
                Type: AcceptMsg,
            }:;
            }
    
            if *targetaddr == "" || *targetnetwork == "" {break}
    
            targetconn, err := net.Dial(*targetnetwork, *targetaddr)
            if err != nil {
                select{
                default:;
                case msg_chan <- ForwardMsg{
                    Type: ErrorMsg,
                    Msg: err,
                }:;
                }
                proxyconn.Close()
                continue
            }
    
            go tcpBridge2(proxyconn,targetconn)
        }
    }(listener, targetaddr, targetnetwork, msg_chan)

    return 
}


func (this *netl) GetLocalDns() error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("nslookup","127.0.0.1")
		output, _ := cmd.CombinedOutput()
		var ip []byte 
		loc_ip := bytes.Index(output,[]byte("Address:"))+8

		for brk:=1;brk>=0; {
			tmp := bytes.IndexAny(output[loc_ip:],"1234567890.")
			if tmp == 0 {
				ip=append(ip,output[loc_ip])
				loc_ip=loc_ip+1
			}else{
				brk=brk-1
				loc_ip=loc_ip+tmp
			}
		}
		
		this.Dns.Server = string(ip)
        return nil
    }else if Checkfile().IsExist("/etc/resolv.conf") {
		cmd := exec.Command("cat","/etc/resolv.conf")
		output, _ := cmd.CombinedOutput()
		var ip []byte 
		loc_ip := bytes.Index(output,[]byte("nameserver"))+10
		for brk:=1;brk>=0; {
			tmp := bytes.IndexAny(output[loc_ip:],"1234567890.")
			if tmp == 0 {
				ip=append(ip,output[loc_ip])
				loc_ip=loc_ip+1
			}else{
				brk=brk-1
				loc_ip=loc_ip+tmp
			}
		}
        this.Dns.Server = string(ip)
        return nil
	}
	Logf().E("[err]Dns: system: ",runtime.GOOS)
	Logf().E("[err]Dns: none")

	return errors.New("1")
}

func MasterDomain(url_s string) (string,error){
    if u,e := url.Parse(url_s);e != nil {
        return "",e
    } else {
        host := u.Hostname()
        list := strings.Split(host, ".")
        if len(list) < 2 {return "",errors.New("invalid domain:"+host)}
        return strings.Join(list[len(list)-2:], "."),nil
    }
    return "",nil
}
