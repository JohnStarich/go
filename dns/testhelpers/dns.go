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

func hostnamesHandler(t *testing.T, ctx context.Context, responseDelay time.Duration, hostnames map[string][]string) dns.HandlerFunc {
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

		switch r.Question[0].Qtype {
		case dns.TypeA:
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

type DNSConfig struct {
	ResponseDelay time.Duration
	Hostnames     map[string][]string
	Port          int
}

func StartDNSServer(t *testing.T, config DNSConfig) (address string, cancel context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	mux := dns.NewServeMux()
	mux.HandleFunc("local.", hostnamesHandler(t, ctx, config.ResponseDelay, config.Hostnames))

	const network = "udp4"
	packetConn, err := net.ListenUDP(network, &net.UDPAddr{Port: config.Port})
	require.NoError(t, err)
	server := &dns.Server{
		Net:        network,
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
	localAddr = "127.0.0.1:" + port
	return localAddr, cancel
}
