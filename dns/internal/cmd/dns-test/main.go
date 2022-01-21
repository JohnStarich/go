package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/johnstarich/go/dns"
	"go.uber.org/zap"
)

func main() {
	flag.Parse()
	hostname := flag.Arg(0)
	if hostname == "" {
		hostname = "api.github.com"
	}
	fmt.Println("Looking up", hostname)

	net.DefaultResolver = dns.NewWithConfig(dns.Config{
		Logger: zap.NewExample(),
	})

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		panic(err)
	}
	fmt.Println(addrs)
}
