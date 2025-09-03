// Command dns-test runs the custom resolver with the given query args.
package main

import (
	"context"
	"flag"
	"fmt"

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

	resolver := dns.NewWithConfig(dns.Config{
		Logger: zap.NewExample(),
	})

	addrs, err := resolver.LookupIPAddr(context.Background(), hostname)
	if err != nil {
		panic(err)
	}
	fmt.Println(addrs)
}
