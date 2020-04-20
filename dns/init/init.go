package init

import (
	"net"

	"github.com/johnstarich/go/dns"
)

func init() {
	net.DefaultResolver = dns.New()
}
