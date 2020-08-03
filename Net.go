package part

import (
	"net"
)

type netl struct{}

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