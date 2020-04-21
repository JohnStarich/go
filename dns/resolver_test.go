package dns

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/johnstarich/go/dns/scutil"
	"github.com/johnstarich/go/dns/staggercast"
	"github.com/johnstarich/go/dns/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var testTimeout = 5 * time.Second

func TestNew(t *testing.T) {
	assert.NotNil(t, New())
}

func TestNewWithConfig(t *testing.T) {
	assert.NotNil(t, NewWithConfig(Config{}))
}

func TestNewWithConfigOS(t *testing.T) {
	t.Run("macOS", func(t *testing.T) {
		resolver := newWithConfig("darwin", Config{})
		require.NotNil(t, resolver)

		assert.True(t, resolver.PreferGo)
		assert.NotNil(t, resolver.Dial)
	})

	t.Run("other", func(t *testing.T) {
		assert.Equal(t, &net.Resolver{}, newWithConfig("linux", Config{}))
	})
}

func TestNewMacOSDialer(t *testing.T) {
	t.Run("fill in defaults", func(t *testing.T) {
		dialer := newMacOSDialer(Config{}).(*macOSDialer)
		assert.Equal(t, zap.NewNop(), dialer.Config.Logger)
		assert.Equal(t, 150*time.Millisecond, dialer.Config.InitialNameserverDelay)
		assert.Equal(t, 10*time.Millisecond, dialer.Config.NextNameserverInterval)
		assert.Equal(t, &net.Dialer{Timeout: 30 * time.Second}, dialer.dialer)
	})

	t.Run("keep settings", func(t *testing.T) {
		someLogger := zap.NewExample()
		someDuration := 36 * time.Millisecond
		dialer := newMacOSDialer(Config{
			Logger:                 someLogger,
			InitialNameserverDelay: someDuration,
			NextNameserverInterval: someDuration,
		}).(*macOSDialer)

		assert.Equal(t, someLogger, dialer.Config.Logger)
		assert.Equal(t, someDuration, dialer.Config.InitialNameserverDelay)
		assert.Equal(t, someDuration, dialer.Config.NextNameserverInterval)
		assert.Equal(t, &net.Dialer{Timeout: 30 * time.Second}, dialer.dialer)
	})
}

func TestEnsureResolvers(t *testing.T) {
	someConfig := scutil.Config{
		Resolvers: []scutil.Resolver{
			{Nameservers: []string{"1.2.3.4"}},
		},
	}
	someError := errors.New("some error")
	dialer := newMacOSDialer(Config{}).(*macOSDialer)
	callCount := 0
	dialer.readResolvers = func(ctx context.Context) (scutil.Config, error) {
		callCount++
		assert.Less(t, callCount, 2, "Read should not be called more than once")
		return someConfig, someError
	}
	resolvers, err := dialer.ensureResolvers()
	assert.Equal(t, someError, err)
	assert.Equal(t, someConfig.Resolvers, resolvers)
	require.Equal(t, someConfig.Resolvers, dialer.resolvers)

	resolvers, err = dialer.ensureResolvers() // should not call read again
	assert.Equal(t, someConfig.Resolvers, resolvers)
	assert.NoError(t, err)
}

func testDialer(t *testing.T) *macOSDialer {
	return newMacOSDialer(Config{Logger: zaptest.NewLogger(t)}).(*macOSDialer)
}
func TestDNSLookupHost(t *testing.T) {
	addr, cancel := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
		ResponseDelay: 1 * time.Second,
		Hostnames: map[string][]string{
			"hi.local.": []string{"5.6.7.8"},
		},
		Port: 53,
	})
	defer cancel()
	host, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.Equal(t, "53", port)
	if host == "::" {
		host = "[::]" // Wrap IPv6 localhost in brackets
	}
	workingDNS := host
	failingDNS := "1.2.3.4"

	for _, tc := range []struct {
		description string
		nameservers []string
		expectErr   string
	}{
		{
			description: "1 working nameserver",
			nameservers: []string{workingDNS},
		},
		{
			description: "1 failing nameserver",
			nameservers: []string{failingDNS},
			expectErr:   "i/o timeout",
		},
		{
			description: "2 working nameservers",
			nameservers: []string{workingDNS, workingDNS},
		},
		{
			description: "2 failing nameservers",
			nameservers: []string{failingDNS, failingDNS},
			expectErr:   "i/o timeout",
		},
		{
			description: "1 failing, 1 working nameserver",
			nameservers: []string{failingDNS, workingDNS},
		},
		{
			description: "1 working, 1 failing nameserver",
			nameservers: []string{workingDNS, failingDNS},
		},
		{
			description: "many failing, 1 working nameserver",
			nameservers: []string{
				failingDNS,
				failingDNS,
				failingDNS,
				failingDNS,
				failingDNS,
				failingDNS,
				failingDNS,
				workingDNS,
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			dialer := testDialer(t)
			dialer.resolvers = []scutil.Resolver{
				{Nameservers: tc.nameservers},
			}

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			conn, err := dialer.DialContext(ctx, "ignored", "ignored")
			require.NoError(t, err)
			assert.Implements(t, (*staggercast.Conn)(nil), conn)
			conn.Close()

			res := &net.Resolver{PreferGo: true, Dial: dialer.DialContext}

			addrs, err := res.LookupHost(ctx, "hi.local")
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, []string{"5.6.7.8"}, addrs)
		})
	}
}
