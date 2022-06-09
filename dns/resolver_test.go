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

const testTimeout = 5 * time.Second

func TestNew(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, New())
}

func TestNewWithConfig(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewWithConfig(Config{}))
}

func TestNewWithConfigOS(t *testing.T) {
	t.Parallel()
	t.Run("macOS", func(t *testing.T) {
		t.Parallel()
		resolver := newWithConfig("darwin", Config{})
		require.NotNil(t, resolver)

		assert.True(t, resolver.PreferGo)
		assert.NotNil(t, resolver.Dial)
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, &net.Resolver{}, newWithConfig("linux", Config{}))
	})
}

func TestNewMacOSDialer(t *testing.T) {
	t.Parallel()
	t.Run("fill in defaults", func(t *testing.T) {
		t.Parallel()
		dialer := newMacOSDialer(Config{})
		assert.Equal(t, zap.NewNop(), dialer.Config.Logger)
		assert.Equal(t, 150*time.Millisecond, dialer.Config.InitialNameserverDelay)
		assert.Equal(t, 10*time.Millisecond, dialer.Config.NextNameserverInterval)
		assert.Equal(t, &net.Dialer{Timeout: 30 * time.Second}, dialer.dialer)
	})

	t.Run("keep settings", func(t *testing.T) {
		t.Parallel()
		someLogger := zap.NewExample()
		someDuration := 36 * time.Millisecond
		dialer := newMacOSDialer(Config{
			Logger:                 someLogger,
			InitialNameserverDelay: someDuration,
			NextNameserverInterval: someDuration,
		})

		assert.Equal(t, someLogger, dialer.Config.Logger)
		assert.Equal(t, someDuration, dialer.Config.InitialNameserverDelay)
		assert.Equal(t, someDuration, dialer.Config.NextNameserverInterval)
		assert.Equal(t, &net.Dialer{Timeout: 30 * time.Second}, dialer.dialer)
	})
}

func TestEnsureNameservers(t *testing.T) {
	t.Parallel()
	someConfig := scutil.Config{
		Resolvers: []scutil.Resolver{
			{Nameservers: []string{
				"1.2.3.4",                      // IPv4
				"fe80::8c86:1eff:fe8b:cc64%5d", // IPv6 with zone ID suffix
			}},
		},
	}
	someError := errors.New("some error")
	dialer := newMacOSDialer(Config{})
	callCount := 0
	dialer.readResolvers = func(ctx context.Context) (scutil.Config, error) {
		callCount++
		assert.Less(t, callCount, 2, "Read should not be called more than once")
		return someConfig, someError
	}
	expectedNameservers := []string{
		"1.2.3.4:53",
		"[fe80::8c86:1eff:fe8b:cc64%5d]:53",
	}
	nameservers, err := dialer.ensureNameservers()
	assert.Equal(t, someError, err)
	assert.Equal(t, expectedNameservers, nameservers)
	require.Equal(t, expectedNameservers, dialer.nameservers)

	nameservers, err = dialer.ensureNameservers() // should not call read again
	assert.Equal(t, expectedNameservers, nameservers)
	assert.NoError(t, err)
}

func testDialer(t *testing.T) *macOSDialer {
	return newMacOSDialer(Config{Logger: zaptest.NewLogger(t)})
}

func TestDNSLookupHost(t *testing.T) {
	t.Parallel()
	const (
		working = true
		failing = false
	)

	for _, tc := range []struct {
		description string
		nameservers []bool // true is working, false is failing as defined above
		expectErr   string
	}{
		{
			description: "1 working nameserver",
			nameservers: []bool{working},
		},
		{
			description: "1 failing nameserver",
			nameservers: []bool{failing},
			expectErr:   "i/o timeout",
		},
		{
			description: "2 working nameservers",
			nameservers: []bool{working, working},
		},
		{
			description: "2 failing nameservers",
			nameservers: []bool{failing, failing},
			expectErr:   "i/o timeout",
		},
		{
			description: "1 failing, 1 working nameserver",
			nameservers: []bool{failing, working},
		},
		{
			description: "1 working, 1 failing nameserver",
			nameservers: []bool{working, failing},
		},
		{
			description: "many failing, 1 working nameserver",
			nameservers: []bool{
				failing,
				failing,
				failing,
				failing,
				failing,
				failing,
				failing,
				working,
			},
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			workingDNS, cancel := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
				ResponseDelay: 1 * time.Second,
				Hostnames: map[string][]string{
					"hi.local.": {"5.6.7.8"},
				},
			})
			defer cancel()
			failingDNS := "1.2.3.4:53"

			dialer := testDialer(t)
			for _, workingNS := range tc.nameservers {
				nameserver := workingDNS
				if !workingNS {
					nameserver = failingDNS
				}
				dialer.nameservers = append(dialer.nameservers, nameserver)
			}

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			conn, err := dialer.DialContext(ctx, "ignored", "ignored")
			require.NoError(t, err)
			assert.IsType(t, (*staggercast.Conn)(nil), conn)
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

func TestDNSLookupHostIPv6(t *testing.T) {
	t.Parallel()
	workingDNS, cancel := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
		ResponseDelay: 1 * time.Second,
		Hostnames: map[string][]string{
			"hi.local.": {"5.6.7.8"},
		},
		Network: "udp6",
	})
	defer cancel()

	dialer := testDialer(t)
	dialer.nameservers = []string{workingDNS}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "ignored", "ignored")
	require.NoError(t, err)
	assert.IsType(t, (*staggercast.Conn)(nil), conn)
	conn.Close()

	res := &net.Resolver{PreferGo: true, Dial: dialer.DialContext}

	addrs, err := res.LookupHost(ctx, "hi.local")
	assert.NoError(t, err)
	assert.Equal(t, []string{"5.6.7.8"}, addrs)
}

func TestReorderNameservers(t *testing.T) {
	t.Parallel()
	addr, cancel := testhelpers.StartDNSServer(t, testhelpers.DNSConfig{
		Hostnames: map[string][]string{
			"hi.local.": {"5.6.7.8"},
		},
	})
	defer cancel()

	const initialDelay = 3 * time.Second
	dialer := newMacOSDialer(Config{
		InitialNameserverDelay: initialDelay,
		Logger:                 zaptest.NewLogger(t),
	})
	dialer.nameservers = []string{"1.2.3.4:53", addr}
	res := &net.Resolver{PreferGo: true, Dial: dialer.DialContext}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	start := time.Now()
	addrs, err := res.LookupHost(ctx, "hi.local")
	duration := time.Since(start)
	cancel() // explicitly cancel to reorder nameservers now

	require.NoError(t, err)
	assert.Equal(t, []string{"5.6.7.8"}, addrs)
	assert.GreaterOrEqual(t, int(duration), int(initialDelay), "First request should take longer than initial delay")
	time.Sleep(time.Second)
	assert.Equal(t, []string{addr, "1.2.3.4:53"}, dialer.nameservers)

	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	start = time.Now()
	addrs, err = res.LookupHost(ctx, "hi.local")
	duration = time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, []string{"5.6.7.8"}, addrs)
	assert.LessOrEqual(t, int(duration), int(initialDelay/100), "Second request should reorder and complete almost instantly")
}

func TestReorderNameserversNoDeadline(t *testing.T) {
	t.Parallel()
	dialer := testDialer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dialer.reorderNameservers(ctx, nil)
	// Reordering should no-op and return immediately
}
