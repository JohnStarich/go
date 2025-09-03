package staggercast

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/johnstarich/go/dns/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 5 * time.Second

func dialUDP(t *testing.T, address string) PacketConn { //nolint:ireturn // Returned interface is a convenience wrapper for tests calling New()
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "udp", address)
	require.NoError(t, err)
	require.Implements(t, (*PacketConn)(nil), conn)
	return conn.(PacketConn)
}

func TestNew(t *testing.T) {
	t.Parallel()

	conns := []PacketConn{
		dialUDP(t, "1.2.3.4:53"),
		dialUDP(t, "5.6.7.8:53"),
	}
	sConn := New(conns)

	assert.Equal(t, conns, sConn.conns)
	assert.Equal(t, uint64(2), sConn.connCount.Load())
	if assert.NotNil(t, sConn.tickerCancel) {
		sConn.tickerCancel()
	}

	require.Len(t, sConn.replay, 2)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	for _, channel := range sConn.replay {
		select {
		case _, open := <-channel:
			assert.False(t, open)
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
	}

	assert.PanicsWithValue(t, "connection count must be non-zero", func() {
		New(nil)
	})
}

func TestDialDNS(t *testing.T) {
	t.Parallel()

	type dnsServer struct {
		delay     time.Duration
		hostnames map[string][]string
	}
	for _, tc := range []struct {
		description string
		servers     []dnsServer
		lookup      string
		expectAddrs []string
		expectErr   string
	}{
		{
			description: "1 working nameserver",
			servers: []dnsServer{
				{hostnames: map[string][]string{
					"hi.local.": {"1.2.3.4"},
				}},
			},
			lookup:      "hi.local",
			expectAddrs: []string{"1.2.3.4"},
		},
		{
			description: "1 unresponsive and 1 working nameserver",
			servers: []dnsServer{
				{delay: 30 * time.Second, hostnames: map[string][]string{
					"hi.local.": {"5.6.7.8"},
				}},
				{hostnames: map[string][]string{
					"hi.local.": {"1.2.3.4"},
				}},
			},
			lookup:      "hi.local",
			expectAddrs: []string{"1.2.3.4"},
		},
		{
			description: "1 unresponsive nameserver",
			servers: []dnsServer{
				{delay: 30 * time.Second, hostnames: map[string][]string{
					"hi.local.": {"5.6.7.8"},
				}},
			},
			lookup:    "hi.local",
			expectErr: "all connections have failed for \"write\": write udp",
		},
		{
			description: "2 unresponsive nameservers",
			servers: []dnsServer{
				{delay: 30 * time.Second, hostnames: map[string][]string{
					"hi.local.": {"1.2.3.4"},
				}},
				{delay: 30 * time.Second, hostnames: map[string][]string{
					"hi.local.": {"5.6.7.8"},
				}},
			},
			lookup:    "hi.local",
			expectErr: "all connections have failed for \"write\": write udp",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			var servers []string
			for _, server := range tc.servers {
				addr := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
					ResponseDelay: server.delay,
					Hostnames:     server.hostnames,
				})
				servers = append(servers, addr)
			}
			t.Logf("DNS servers, in-order: %+v", servers)

			res := &net.Resolver{
				PreferGo: true,
				Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
					var conns []PacketConn
					for _, addr := range servers {
						conns = append(conns, dialUDP(t, addr))
					}
					return New(conns), nil
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			addrs, err := res.LookupHost(ctx, tc.lookup)
			if tc.expectErr != "" {
				assert.Equal(t, tc.expectAddrs, addrs)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectAddrs, addrs)
		})
	}
}

func TestStagger(t *testing.T) {
	t.Parallel()

	startServers := func(t *testing.T) []string {
		const slowServerDelay = 30 * time.Second
		var servers []string
		addr := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
			ResponseDelay: slowServerDelay,
			Hostnames: map[string][]string{
				"hi.local.": {"1.2.3.4"},
			},
		})
		servers = append(servers, addr)
		addr = testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
			Hostnames: map[string][]string{
				"hi.local.": {"5.6.7.8"},
			},
		})
		servers = append(servers, addr)
		return servers
	}

	t.Run("stagger never enables", func(t *testing.T) {
		t.Parallel()
		servers := startServers(t)
		res := &net.Resolver{
			PreferGo: true,
			Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
				var conns []PacketConn
				for _, addr := range servers {
					conns = append(conns, dialUDP(t, addr))
				}
				conn := New(conns)
				conn.Stagger(nil, func() {}) // effectively disable all further connections
				return conn, nil
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		addrs, err := res.LookupHost(ctx, "hi.local")
		assert.Empty(t, addrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all connections have failed for \"write\": write udp")
	})

	t.Run("stagger eventually succeeds", func(t *testing.T) {
		t.Parallel()
		if os.Getenv("CI") == "true" && runtime.GOOS == "darwin" {
			t.Skip("DNS timeouts on macOS in CI don't work very well.") // FIXME: This doesn't seem to work on macOS in CI, despite working fine locally.
		}

		servers := startServers(t)
		const delay = 1 * time.Second
		res := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var conns []PacketConn
				for _, addr := range servers {
					conns = append(conns, dialUDP(t, addr))
				}
				conn := New(conns)

				// stagger new connections with a ticker of 'delay' interval
				ctx, cancel := context.WithCancel(ctx)
				c := make(chan struct{})
				go func() {
					ticker := time.NewTicker(delay)
					for {
						select {
						case <-ticker.C:
							c <- struct{}{}
						case <-ctx.Done():
							ticker.Stop()
							return
						}
					}
				}()
				conn.Stagger(c, cancel) // effectively disable all further connections
				return conn, nil
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		start := time.Now()
		addrs, err := res.LookupHost(ctx, "hi.local")
		d := time.Since(start)
		assert.NoError(t, err)
		assert.Equal(t, []string{"5.6.7.8"}, addrs)
		assert.LessOrEqual(t, int(delay), int(d), "DNS should not resolve before second connection is enabled")
		assert.GreaterOrEqual(t, int(testTimeout), int(d), "DNS should resolve before the test times out")
	})

	t.Run("stagger enables all (almost) instantly", func(t *testing.T) {
		t.Parallel()
		servers := startServers(t)
		res := &net.Resolver{
			PreferGo: true,
			Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
				var conns []PacketConn
				for _, addr := range servers {
					conns = append(conns, dialUDP(t, addr))
				}
				conn := New(conns)

				// stagger new connections with a ticker of 'delay' interval
				c := make(chan struct{})
				close(c)
				conn.Stagger(c, func() {}) // effectively disable all further connections
				return conn, nil
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		start := time.Now()
		addrs, err := res.LookupHost(ctx, "hi.local")
		d := time.Since(start)
		assert.NoError(t, err)
		assert.Equal(t, []string{"5.6.7.8"}, addrs)
		assert.Less(t, int(d), int(float64(testTimeout)*0.01), "Second connection should be used almost immediately")
	})
}

func TestLocalAddr(t *testing.T) {
	t.Parallel()
	firstConn := dialUDP(t, "1.2.3.4:53")
	conn := New([]PacketConn{
		firstConn,
		dialUDP(t, "5.6.7.8:53"),
	})
	assert.Equal(t, firstConn.LocalAddr(), conn.LocalAddr())
}

func TestRemoteAddr(t *testing.T) {
	t.Parallel()
	firstConn := dialUDP(t, "1.2.3.4:53")
	conn := New([]PacketConn{
		firstConn,
		dialUDP(t, "5.6.7.8:53"),
	})
	assert.Equal(t, firstConn.RemoteAddr(), conn.RemoteAddr())
}

type wrapperConn struct {
	PacketConn
	readDeadline, writeDeadline time.Time
	remoteAddr                  net.Addr
}

func (w *wrapperConn) SetWriteDeadline(t time.Time) error {
	w.writeDeadline = t
	return nil
}

func (w *wrapperConn) SetReadDeadline(t time.Time) error {
	w.readDeadline = t
	return nil
}

func (w *wrapperConn) RemoteAddr() net.Addr {
	return w.remoteAddr
}

func TestSetReadDeadline(t *testing.T) {
	t.Parallel()
	conns := []PacketConn{
		&wrapperConn{},
		&wrapperConn{},
	}
	conn := New(conns)
	someTime := time.Now()
	assert.NoError(t, conn.SetReadDeadline(someTime))
	for _, conn := range conns {
		assert.Equal(t, someTime, conn.(*wrapperConn).readDeadline)
	}
}

func TestSetWriteDeadline(t *testing.T) {
	t.Parallel()
	conns := []PacketConn{
		&wrapperConn{},
		&wrapperConn{},
	}
	conn := New(conns)
	someTime := time.Now()
	assert.NoError(t, conn.SetWriteDeadline(someTime))
	for _, conn := range conns {
		assert.Equal(t, someTime, conn.(*wrapperConn).writeDeadline)
	}
}

func TestStats(t *testing.T) {
	t.Parallel()
	_, addr, err := net.ParseCIDR("192.0.2.0/24")
	require.NoError(t, err)

	conn1, conn2 := &wrapperConn{}, &wrapperConn{remoteAddr: addr}

	conn := New([]PacketConn{conn1, conn2})
	conn.firstResponder.Store(1)
	stats := conn.Stats()
	assert.Equal(t, Stats{
		FastestRemoteIndex: 1,
		FastestRemote:      addr,
	}, stats)
}
