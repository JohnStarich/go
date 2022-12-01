// Package init provides a convenient side-effecting import to set the custom DNS resolver.
package init

import (
	"net"

	"github.com/johnstarich/go/dns"
)

func init() { //nolint:gochecknoinits // This package's purpose is an implicit init().
	net.DefaultResolver = dns.New()
}
