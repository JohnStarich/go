package testhelpers

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

const dom = "dns.local."

func hostnamesHandler(ctx context.Context, t *testing.T, responseDelay time.Duration, hostnames map[string][]string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		timer := time.NewTimer(responseDelay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}

		var message dns.Msg
		message.SetReply(r)

		if r.Question[0].Qtype == dns.TypeA {
			hostname := r.Question[0].Name
			for _, ip := range hostnames[hostname] {
				message.Answer = append(message.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: dom, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
					A:   net.ParseIP(ip).To4(),
				})
			}
		}

		t.Logf("DNS message:\n%s", message.String())
		err := w.WriteMsg(&message)
		if err != nil {
			t.Logf("Error writing message: %s", err.Error())
		}
	}
}

// DNSConfig contains options to configure a test DNS server from StartDNSServer()
type DNSConfig struct {
	ResponseDelay time.Duration
	Hostnames     map[string][]string
	Port          int
	Network       string // e.g. udp4, udp6
}

// StartDNSServer starts a DNS server with the given configuration
func StartDNSServer(t *testing.T, config DNSConfig) (address string, cancel context.CancelFunc) {
	t.Helper()
	const (
		netUDP4 = "udp4"
		netUDP6 = "udp6"
	)
	if config.Network == "" {
		config.Network = netUDP4
	}
	switch config.Network {
	case netUDP4, netUDP6:
	default:
		t.Fatal("Unsupported network for DNS:", config.Network)
	}

	ctx, cancel := context.WithCancel(context.Background())
	mux := dns.NewServeMux()
	mux.HandleFunc("local.", hostnamesHandler(ctx, t, config.ResponseDelay, config.Hostnames))

	packetConn, err := net.ListenUDP(config.Network, &net.UDPAddr{Port: config.Port})
	require.NoError(t, err)
	server := &dns.Server{
		Net:        config.Network,
		PacketConn: packetConn,
		Handler:    mux,
	}
	go func() {
		err := server.ActivateAndServe()
		require.NoError(t, err)
	}()
	go func() {
		<-ctx.Done()
		_ = server.Shutdown()
	}()
	localAddr := server.PacketConn.LocalAddr().String()
	_, port, err := net.SplitHostPort(localAddr)
	require.NoError(t, err)
	switch config.Network {
	case netUDP4:
		localAddr = "127.0.0.1:" + port
	case netUDP6:
		localAddr = "[::1]:" + port
	}
	return localAddr, cancel
}
