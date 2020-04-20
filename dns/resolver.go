package dns

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/johnstarich/go/dns/scutil"
	"github.com/johnstarich/go/dns/staggercast"
	"go.uber.org/zap"
)

type Config struct {
	Logger *zap.Logger
}

const macOSRuntimeName = "darwin"

func New() *net.Resolver {
	return NewWithConfig(Config{})
}

func NewWithConfig(config Config) *net.Resolver {
	if runtime.GOOS != macOSRuntimeName {
		return &net.Resolver{}
	}

	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	dialer := newMacOSDialer(config)

	return &net.Resolver{
		PreferGo: true,
		Dial:     dialer.DialContext,
	}
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type macOSDialer struct {
	Config
	dialer    *net.Dialer
	resolvers []scutil.Resolver
}

func newMacOSDialer(config Config) Dialer {
	return &macOSDialer{
		dialer: &net.Dialer{Timeout: 30 * time.Second},
	}
}

func (m *macOSDialer) ensureResolvers() ([]scutil.Resolver, error) {
	if len(m.resolvers) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cfg, err := scutil.ReadMacOSDNS(ctx)
		m.resolvers = cfg.Resolvers
		return m.resolvers, err
	}
	return m.resolvers, nil
}

func (m *macOSDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	resolvers, err := m.ensureResolvers()
	if err != nil {
		m.Logger.Error("Failed looking up macOS resolvers, falling back to builtin DNS dialer", zap.Error(err))
		return m.dialer.DialContext(ctx, network, address)
	}

	var nameservers []string
	for _, resolver := range resolvers {
		nameservers = append(nameservers, resolver.Nameservers...)
	}

	conn, err := m.dialAll(ctx, nameservers)
	if err != nil {
		m.Logger.Error("Failed dialing macOS nameservers, falling back to builtin DNS dialer", zap.Error(err))
		return m.dialer.DialContext(ctx, network, address)
	}

	return conn, nil
}

func (m *macOSDialer) dialAll(ctx context.Context, nameservers []string) (net.Conn, error) {
	conns := make(chan net.Conn, len(nameservers))

	var wait sync.WaitGroup
	wait.Add(len(nameservers))

	for _, nameserver := range nameservers {
		go func(nameserver string) {
			conn, err := m.dialer.DialContext(ctx, "udp", nameserver+":53")
			if err != nil {
				m.Logger.Warn("Error dialing nameserver", zap.String("nameserver", nameserver), zap.Error(err))
			} else {
				conns <- conn
			}
			wait.Done()
		}(nameserver)
	}
	wait.Wait()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	close(conns)

	var allSuccessfulConns []staggercast.Conn
	for conn := range conns {
		allSuccessfulConns = append(allSuccessfulConns, conn.(staggercast.Conn)) // UDP connections must also support net.PacketConn
	}
	if len(allSuccessfulConns) == 0 {
		return nil, errors.New("error dialing all nameservers")
	}
	return staggercast.New(allSuccessfulConns), nil
}
