package part

import (
    "net"
    "sync"
    "errors"
	"bytes"
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

func (t *netl) Forward(targetaddr,targetnetwork *string, listenaddr string,Need_Accept bool) {
    proxylistener, err := net.Listen("tcp", listenaddr + ":0")
    if err != nil {
        Logf().E("[part/Forward]Unable to listen, error:", err.Error())
    }
    const max = 1000
    var accept_chan chan bool = make(chan bool,max)
    t.RV = append(t.RV,proxylistener.Addr().(*net.TCPAddr).Port,accept_chan,err)

    defer proxylistener.Close()

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

        proxyconn, err := proxylistener.Accept()
        if err != nil {
            Logf().E("[part/Forward]Unable to accept a request:", err.Error())
            continue
        }
        
        if Need_Accept {
            if len(accept_chan) == max {
                Logf().E("[part/Forward] accept channel full.Skip")
                <- accept_chan 
            }
            accept_chan <- true
        }
        if *targetaddr == "" || *targetnetwork == "" {
            proxyconn.Close()
            Logf().I("[part/Forward]Stop!", *targetaddr, *targetnetwork)
            break
        }

        retry := 0
        for {
            targetconn, err := net.Dial(*targetnetwork, *targetaddr)
            if err != nil {
                Logf().E("[part/Forward]Unable to connect:", *targetaddr, err.Error())
                retry += 1
                if retry >= 2 {proxyconn.Close();break}
                time.Sleep(time.Duration(1)*time.Millisecond)
                continue
            }    

            go tcpBridge2(proxyconn,targetconn)
            break
        }
    }

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
