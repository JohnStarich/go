package dns

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sort"
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
	type dialResp struct {
		ix   int
		conn net.Conn
	}
	conns := make(chan dialResp, len(nameservers))

	var wait sync.WaitGroup
	wait.Add(len(nameservers))

	for ix, nameserver := range nameservers {
		go func(ix int, nameserver string) {
			conn, err := m.dialer.DialContext(ctx, "udp", nameserver+":53")
			if err != nil {
				m.Logger.Warn("Error dialing nameserver", zap.String("nameserver", nameserver), zap.Error(err))
			} else {
				conns <- dialResp{ix: ix, conn: conn}
			}
			wait.Done()
		}(ix, nameserver)
	}
	wait.Wait()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	close(conns)

	var allSuccessfulResps []dialResp
	for resp := range conns {
		allSuccessfulResps = append(allSuccessfulResps, resp)
	}
	if len(allSuccessfulResps) == 0 {
		return nil, errors.New("error dialing all nameservers")
	}
	sort.Slice(allSuccessfulResps, func(a, b int) bool {
		return allSuccessfulResps[a].ix < allSuccessfulResps[b].ix
	})

	var allSuccessfulConns []staggercast.PacketConn
	for _, resp := range allSuccessfulResps {
		allSuccessfulConns = append(allSuccessfulConns, resp.conn.(staggercast.PacketConn)) // UDP connections must also support net.PacketConn
	}

	staggerConn := staggercast.New(allSuccessfulConns)
	staggerConn.Stagger(staggerTicker(150*time.Millisecond, 10*time.Millisecond))
	return staggerConn, nil
}

func staggerTicker(initialDelay, d time.Duration) (<-chan struct{}, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan struct{})
	go func() {
		timer := time.NewTimer(initialDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		timer.Stop()

		ticker := time.NewTicker(d)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				close(c)
				return
			case <-ticker.C:
				c <- struct{}{}
			}
		}
	}()
	return c, cancel
}
