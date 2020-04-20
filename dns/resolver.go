package dns

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/johnstarich/go/dns/scutil"
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
		return nil, err
	}
	count := 0
	for _, resolver := range resolvers {
		count += len(resolver.Nameservers)
	}
	conns := make(chan net.Conn, count)

	nsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, resolver := range resolvers {
		for _, nameserver := range resolver.Nameservers {
			go func(nameserver string) {
				conn, err := m.dialer.DialContext(nsCtx, "udp", nameserver+":53")
				if err != nil {
					m.Logger.Warn("Error dialing nameserver", zap.String("nameserver", nameserver), zap.Error(err))
				} else {
					conns <- conn
				}
			}(nameserver)
		}
	}
	select {
	case conn := <-conns:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-nsCtx.Done():
	}
	m.Logger.Error("Falling back to builtin DNS dialer")
	return m.dialer.DialContext(ctx, network, address)
}
