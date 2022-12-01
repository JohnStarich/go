// Package dns implements a working macOS DNS resolver (really just a `Dial`er) for projects that must cross-compile from Linux systems or just don't want CGO.
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/johnstarich/go/dns/scutil"
	"github.com/johnstarich/go/dns/staggercast"
	"go.uber.org/zap"
)

// Config contains options for configuring a DNS resolver.
// Currently has no effect on non-macOS runtimes.
type Config struct {
	Logger                 *zap.Logger
	InitialNameserverDelay time.Duration
	NextNameserverInterval time.Duration
}

const macOSRuntimeName = "darwin"

// New returns a default DNS resolver for this runtime
func New() *net.Resolver {
	return NewWithConfig(Config{})
}

// NewWithConfig returns a DNS resolver with the given config for this runtime
func NewWithConfig(config Config) *net.Resolver {
	return newWithConfig(runtime.GOOS, config)
}

func newWithConfig(runtimeName string, config Config) *net.Resolver {
	if runtimeName != macOSRuntimeName {
		return &net.Resolver{}
	}

	dialer := newMacOSDialer(config)
	return &net.Resolver{
		PreferGo: true,
		Dial:     dialer.DialContext,
	}
}

type macOSDialer struct {
	Config
	dialer        *net.Dialer
	nameservers   []string
	nameserversMu sync.RWMutex

	readResolvers func(context.Context) (scutil.Config, error)
}

func newMacOSDialer(config Config) *macOSDialer {
	const (
		defaultInitialNameserverDelay = 150 * time.Millisecond
		defaultNextNameserverInterval = 10 * time.Millisecond
		maxTimeout                    = 30 * time.Second
	)
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.InitialNameserverDelay == 0 {
		config.InitialNameserverDelay = defaultInitialNameserverDelay
	}
	if config.NextNameserverInterval == 0 {
		config.NextNameserverInterval = defaultNextNameserverInterval
	}

	return &macOSDialer{
		Config:        config,
		dialer:        &net.Dialer{Timeout: maxTimeout},
		readResolvers: scutil.ReadMacOSDNS,
	}
}

func (m *macOSDialer) ensureNameservers() ([]string, error) {
	m.nameserversMu.RLock()
	nameservers := m.nameservers
	m.nameserversMu.RUnlock()
	if len(nameservers) != 0 {
		return nameservers, nil
	}
	m.nameserversMu.Lock()
	defer m.nameserversMu.Unlock()
	if len(m.nameservers) != 0 { // check again, could change while waiting on lock
		return m.nameservers, nil
	}

	const maxTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), maxTimeout)
	defer cancel()
	m.Logger.Info("Reading macOS DNS config from 'scutil'...")
	cfg, err := m.readResolvers(ctx)
	for _, resolver := range cfg.Resolvers {
		port := resolver.Port
		if port == 0 {
			port = 53
		}
		for _, nameserver := range resolver.Nameservers {
			m.nameservers = append(m.nameservers, dialableNameserver(nameserver)+":"+fmt.Sprint(port))
		}
	}
	m.Logger.Info("Finished reading macOS DNS config from 'scutil'", zap.Error(err))
	return m.nameservers, err
}

func (m *macOSDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	nameservers, err := m.ensureNameservers()
	if err != nil {
		m.Logger.Error("Failed looking up macOS resolvers, falling back to builtin DNS dialer", zap.Error(err))
		return m.dialer.DialContext(ctx, network, address)
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

	m.Logger.Debug("Dialing all nameservers")
	for ix, nameserver := range nameservers {
		go func(ix int, nameserver string) {
			conn, err := m.dialer.DialContext(ctx, "udp", nameserver)
			if err != nil {
				m.Logger.Warn("Error dialing nameserver", zap.String("nameserver", nameserver), zap.Int("index", ix), zap.Error(err))
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
	sort.Slice(allSuccessfulResps, func(a, b int) bool { // preserve original nameserver order
		return allSuccessfulResps[a].ix < allSuccessfulResps[b].ix
	})

	var allSuccessfulConns []staggercast.PacketConn
	for _, resp := range allSuccessfulResps {
		allSuccessfulConns = append(allSuccessfulConns, resp.conn.(staggercast.PacketConn)) // UDP connections must also support net.PacketConn
	}

	staggerConn := staggercast.New(allSuccessfulConns)
	staggerConn.Stagger(staggerTicker(m.InitialNameserverDelay, m.NextNameserverInterval, m.Logger))
	go m.reorderNameservers(ctx, staggerConn)
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

func (m *macOSDialer) reorderNameservers(ctx context.Context, conn *staggercast.Conn) {
	var zero time.Time
	if deadline, ok := ctx.Deadline(); !ok || deadline == zero {
		m.Logger.Debug("Skipping nameserver reorder, no deadline on context")
		return
	}

	<-ctx.Done()
	stats := conn.Stats()
	if stats.FastestRemoteIndex == 0 {
		return
	}

	m.nameserversMu.Lock()
	defer m.nameserversMu.Unlock()

	fastestRemoteAddr := stats.FastestRemote.String()
	fastIndexNS := m.nameservers[stats.FastestRemoteIndex]
	if fastIndexNS != fastestRemoteAddr {
		m.Logger.Debug("Fastest remote already reordered", zap.String("remote", stats.FastestRemote.String()), zap.String("nsIndex", fastIndexNS))
		return
	}
	newNS := []string{fastIndexNS}
	newNS = append(newNS, m.nameservers[:stats.FastestRemoteIndex]...)
	newNS = append(newNS, m.nameservers[stats.FastestRemoteIndex+1:]...)
	m.nameservers = newNS
	m.Logger.Debug("Reordering fastest nameserver to the front", zap.Strings("nameservers", newNS))
}

func dialableNameserver(nameserver string) string {
	ipStr := nameserver
	if i := strings.IndexRune(nameserver, '%'); i != -1 {
		// remove optional IPv6 zone ID suffix to parse IP
		ipStr = nameserver[:i]
	}
	isIPv4 := net.ParseIP(ipStr).To4() != nil
	if !isIPv4 {
		return fmt.Sprintf("[%s]", nameserver)
	}
	return nameserver
}
