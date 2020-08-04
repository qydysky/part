package part

import (
    "net"
    "sync"
    // "errors"
)

type netl struct{
    RV []interface{}
}

func Net() *netl{
	return &netl{}
}

func (*netl) TestDial(network,address string) bool {
    conn, err := net.Dial(network,address)
    if err != nil {
		Logf().E(err.Error())
        return false
    }
    conn.Close()
    return true
}

func (t *netl) Forward(targetaddr,targetnetwork *string, Need_Accept bool) {
    proxylistener, err := net.Listen("tcp", "127.0.0.1:0")
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

		targetconn, err := net.Dial(*targetnetwork, *targetaddr)
        if err != nil {
            Logf().E("[part/Forward]Unable to connect:", *targetaddr, err.Error())
            proxyconn.Close()
            continue
        }
        
        go tcpBridge2(proxyconn,targetconn)
    }

}