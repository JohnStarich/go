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
	Logger                 *zap.Logger
	InitialNameserverDelay time.Duration
	NextNameserverInterval time.Duration
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
	if config.InitialNameserverDelay == 0 {
		config.InitialNameserverDelay = 150 * time.Millisecond
	}
	if config.NextNameserverInterval == 0 {
		config.NextNameserverInterval = 10 * time.Millisecond
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
	dialer      *net.Dialer
	resolvers   []scutil.Resolver
	resolversMu sync.Mutex
}

func newMacOSDialer(config Config) Dialer {
	return &macOSDialer{
		Config: config,
		dialer: &net.Dialer{Timeout: 30 * time.Second},
	}
}

func (m *macOSDialer) ensureResolvers() ([]scutil.Resolver, error) {
	if len(m.resolvers) != 0 {
		return m.resolvers, nil
	}
	m.resolversMu.Lock()
	defer m.resolversMu.Unlock()
	if len(m.resolvers) != 0 { // check again, could change while waiting on lock
		return m.resolvers, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.Logger.Info("Reading macOS DNS config from 'scutil'...")
	cfg, err := scutil.ReadMacOSDNS(ctx)
	m.resolvers = cfg.Resolvers
	m.Logger.Info("Finished reading macOS DNS config from 'scutil'", zap.Error(err))
	return m.resolvers, err
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
			logger := m.Logger.With(zap.String("nameserver", nameserver), zap.Int("index", ix))
			logger.Debug("Dialing nameserver...")
			conn, err := m.dialer.DialContext(ctx, "udp", nameserver+":53")
			if err != nil {
				logger.Warn("Error dialing nameserver", zap.Error(err))
			} else {
				logger.Debug("Dial succeeded")
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
	sort.Slice(allSuccessfulResps, func(a, b int) bool { // preserve original nameserver order
		return allSuccessfulResps[a].ix < allSuccessfulResps[b].ix
	})

	var allSuccessfulConns []staggercast.PacketConn
	for _, resp := range allSuccessfulResps {
		allSuccessfulConns = append(allSuccessfulConns, resp.conn.(staggercast.PacketConn)) // UDP connections must also support net.PacketConn
	}

	staggerConn := staggercast.New(allSuccessfulConns)
	staggerConn.Stagger(staggerTicker(m.InitialNameserverDelay, m.NextNameserverInterval, m.Logger))
	return staggerConn, nil
}

func staggerTicker(initialDelay, d time.Duration, logger *zap.Logger) (<-chan struct{}, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan struct{})
	go func() {
		index := 1
		timer := time.NewTimer(initialDelay)
		select {
		case <-timer.C:
			logger.Debug("scatter: Enabling connection", zap.Int("index", index))
			index++
			c <- struct{}{}
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
				logger.Debug("scatter: Enabling connection", zap.Int("index", index))
				index++
				c <- struct{}{}
			}
		}
	}()
	return c, cancel
}
