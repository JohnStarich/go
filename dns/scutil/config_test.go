package scutil

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealSCUtilDNS(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("scutil only available on macOS")
	}

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		_, err := ReadMacOSDNS(context.Background())
		assert.NoError(t, err)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := ReadMacOSDNS(ctx)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestReadMacOSDNS(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description     string
		scutilOutput    string
		scutilErr       error
		expectResolvers []Resolver
		expectErr       bool
	}{
		{
			description: "success, no output",
		},
		{
			description: "scutil failed",
			scutilErr:   errors.New("some error"),
			expectErr:   true,
		},
		{
			description: "one nameserver",
			scutilOutput: `
resolver #1
  nameserver[0] : 1.2.3.4
			`,
			expectResolvers: []Resolver{
				{
					Nameservers: []string{"1.2.3.4"},
				},
			},
		},
		{
			description: "ignore fields before resolver",
			scutilOutput: `
  nameserver[0] : 0.0.0.0
resolver #1
  nameserver[0] : 1.2.3.4
			`,
			expectResolvers: []Resolver{
				{
					Nameservers: []string{"1.2.3.4"},
				},
			},
		},
		{
			description: "one full resolver",
			scutilOutput: `
resolver #1
  nameserver[0] : 1.2.3.4
  nameserver[1] : 5.6.7.8
  domain   : local
  search domain[0] : search1
  search domain[1] : search2
  options  : mdns
  timeout  : 5
  if_index : 1 (en0)
  flags    : Scoped, Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)
  order    : 300000
			`,
			expectResolvers: []Resolver{
				{
					Domain:         "local",
					Flags:          []Flag{Scoped, RequestARecords},
					InterfaceIndex: 1,
					InterfaceName:  "en0",
					MulticastDNS:   true,
					Nameservers:    []string{"1.2.3.4", "5.6.7.8"},
					Order:          300000,
					Reach:          []Reach{Reachable, DirectlyReachableAddress},
					reachable:      true,
					SearchDomain:   []string{"search1", "search2"},
					Timeout:        5 * time.Second,
				},
			},
		},
		{
			description: "one full IPv6 nameserver",
			scutilOutput: `
resolver #1
  nameserver[0] : fe80::8c86:1eff:fe8b:cc64%5d
  nameserver[1] : 172.20.10.1
  if_index : 5 (en0)
  flags    : Request A records, Request AAAA records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)
			`,
			expectResolvers: []Resolver{
				{
					Flags:          []Flag{RequestARecords, RequestAAAARecords},
					Nameservers:    []string{"fe80::8c86:1eff:fe8b:cc64%5d", "172.20.10.1"},
					InterfaceIndex: 5,
					InterfaceName:  "en0",
					Reach:          []Reach{Reachable, DirectlyReachableAddress},
					reachable:      true,
				},
			},
		},
		{
			description: "full output example",
			scutilOutput: `
DNS configuration

resolver #1
  nameserver[0] : 8.8.8.8
  nameserver[1] : 8.8.4.4
  if_index : 5 (en0)
  flags    : Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)

resolver #2
  domain   : local
  options  : mdns
  timeout  : 5
  flags    : Request A records
  reach    : 0x00000000 (Not Reachable)
  order    : 300000

resolver #3
  domain   : 9.e.f.ip6.arpa
  options  : mdns
  timeout  : 5
  flags    : Request A records
  reach    : 0x00000000 (Not Reachable)
  order    : 300600

resolver #4
  domain   : 10.in-addr.arpa
  nameserver[0] : 127.0.0.1
  port     : 8600
  timeout  : 5
  flags    : Request A records, Request AAAA records
  reach    : 0x00030002 (Reachable,Local Address,Directly Reachable Address)

DNS configuration (for scoped queries)

resolver #1
  nameserver[0] : 8.8.8.8
  nameserver[1] : 8.8.4.4
  if_index : 5 (en0)
  flags    : Scoped, Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)

resolver #2
  nameserver[0] : 8.8.8.8
  nameserver[1] : 8.8.4.4
  if_index : 10 (en10)
  flags    : Scoped, Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)
			`,
			expectResolvers: []Resolver{
				{
					Nameservers:    []string{"8.8.8.8", "8.8.4.4"},
					InterfaceIndex: 5,
					InterfaceName:  "en0",
					Flags:          []Flag{RequestARecords},
					Reach:          []Reach{Reachable, DirectlyReachableAddress},
					reachable:      true,
				},
				{
					Domain:       "local",
					MulticastDNS: true,
					Timeout:      5 * time.Second,
					Flags:        []Flag{RequestARecords},
					Reach:        []Reach{NotReachable},
					Order:        300000,
				},
				{
					Domain:       "9.e.f.ip6.arpa",
					MulticastDNS: true,
					Timeout:      5 * time.Second,
					Flags:        []Flag{RequestARecords},
					Reach:        []Reach{NotReachable},
					Order:        300600,
				},
				{
					Domain:      "10.in-addr.arpa",
					Port:        8600,
					Nameservers: []string{"127.0.0.1"},
					Timeout:     5 * time.Second,
					Flags:       []Flag{RequestARecords, RequestAAAARecords},
					Reach:       []Reach{Reachable, LocalAddress, DirectlyReachableAddress},
					reachable:   true,
				},
				{
					Nameservers:    []string{"8.8.8.8", "8.8.4.4"},
					InterfaceIndex: 5,
					InterfaceName:  "en0",
					Flags:          []Flag{Scoped, RequestARecords},
					Reach:          []Reach{Reachable, DirectlyReachableAddress},
					reachable:      true,
				},
				{
					Nameservers:    []string{"8.8.8.8", "8.8.4.4"},
					InterfaceIndex: 10,
					InterfaceName:  "en10",
					Flags:          []Flag{Scoped, RequestARecords},
					Reach:          []Reach{Reachable, DirectlyReachableAddress},
					reachable:      true,
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			getSCUtilDNS := func(context.Context) ([]byte, error) {
				return []byte(tc.scutilOutput), tc.scutilErr
			}
			cfg, err := readMacOSDNS(ctx, getSCUtilDNS)
			if tc.expectErr {
				assert.Equal(t, tc.scutilErr, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expectResolvers, cfg.Resolvers)
		})
	}
}

func TestReachable(t *testing.T) {
	t.Parallel()
	assert.False(t, Resolver{reachable: false}.Reachable())
	assert.True(t, Resolver{reachable: true}.Reachable())
}
