package part

import (
	"testing"
)

func Test_customMap(t *testing.T) {
	t.Log(GetIntranetIp(`192.168.0.0/16`))
	t.Log(GetIpByCidr())
}
