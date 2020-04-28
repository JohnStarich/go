package staggercast

// Staggercast implements a one-to-many net.Conn for easy scattering of the same request to multiple endpoints with control on when new connections are attempted.
// For ease of use in a DNS resolver Dial func, all connections implement both net.Conn and net.PacketConn.

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

type Conn interface {
	PacketConn
	Stagger(ticker <-chan struct{}, cancel context.CancelFunc)
	Stats() Stats
}

// PacketConn implements both net.Conn and net.PacketConn
type PacketConn interface {
	io.Reader
	io.Writer
	RemoteAddr() net.Addr
	net.PacketConn
}

type Stats struct {
	FastestRemoteIndex int
	FastestRemote      net.Addr
}

// staggerConn fires out all Writes to all outgoing connections, Reads return the first successful read.
type staggerConn struct {
	conns []PacketConn

	connCount    *atomic.Uint64
	replay       []chan struct{}
	replayMu     sync.RWMutex
	tickerCancel context.CancelFunc
	// capture last Write and SetDeadlines for replay on staggered connections
	lastDeadline,
	lastReadDeadline,
	lastWriteDeadline time.Time
	lastWrite     []byte
	lastWriteAddr net.Addr // currently unused in replay

	// capture stats about connections for better ordering on the next call to staggercast.New
	firstResponder *atomic.Int64
}

func New(conns []PacketConn) Conn {
	if len(conns) == 0 {
		panic("connection count must be non-zero")
	}
	replay := make([]chan struct{}, len(conns))
	for i := range replay {
		replay[i] = make(chan struct{})
		close(replay[i])
	}
	return &staggerConn{
		conns: conns,

		// by default, fire all connections at once
		connCount:    atomic.NewUint64(uint64(len(conns))),
		replay:       replay,
		tickerCancel: func() {},

		firstResponder: atomic.NewInt64(-1), // -1 marks no known first responder, so connection #0 can be designated first
	}
}

// Stagger controls when secondary connections are attempted after the first. For any new enabled connection, the last Write and SetDeadlines are replayed onto it.
// If Stagger is called after the connection is started, then the behavior is undefined.
// First connection is attempted immediately, then all future connections will be enabled when receiving from ticker.
//
// Example: Create and wrap a 'time.Ticker.C' with a duration of 1 second.
// The first connection is attempted immediately. For every 1 second the Read hasn't returned, an additional connection is attempted.
// Once a connection succeeds, the result is returned immediately from Read.
func (s *staggerConn) Stagger(ticker <-chan struct{}, cancel context.CancelFunc) {
	s.replayMu.Lock()

	ctx, tickerCancel := context.WithCancel(context.Background())
	s.tickerCancel = tickerCancel

	totalLength := len(s.conns)
	s.replay = make([]chan struct{}, totalLength)
	for i := range s.replay {
		s.replay[i] = make(chan struct{})
	}
	close(s.replay[0]) // initial connection should immediately fire
	connCount := atomic.NewUint64(1)
	s.connCount = connCount

	// listen for ticks and fire off replays on new connections
	go func() {
		// only reference params and local vars here, no state from 'p'
		for {
			select {
			case <-ticker:
				count := connCount.Inc()
				if count > uint64(totalLength) {
					// finished enabling all connections
					cancel()
					return
				}
				go s.runReplay(count - 1)
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()
	s.replayMu.Unlock()
}

func (s *staggerConn) Stats() Stats {
	ix := int(s.firstResponder.Load())
	if ix < 0 {
		ix = 0 // firstResponder of 0 is a no-op if re-order should occur
	}
	return Stats{
		FastestRemoteIndex: ix,
		FastestRemote:      s.conns[ix].RemoteAddr(),
	}
}

type connOp string

const (
	readOp             connOp = "read"
	writeOp            connOp = "write"
	readFromOp         connOp = "read from"
	writeToOp          connOp = "write to"
	setDeadlineOp      connOp = "set deadline"
	setReadDeadlineOp  connOp = "set read deadline"
	setWriteDeadlineOp connOp = "set write deadline"
)

// runReplay synchronously reapplies the last Deadlines and the last Write on a best-effort basis
func (s *staggerConn) runReplay(connIndex uint64) {
	s.replayMu.RLock()
	if !s.lastDeadline.IsZero() {
		_ = s.conns[connIndex].SetDeadline(s.lastDeadline)
	}
	if !s.lastReadDeadline.IsZero() {
		_ = s.conns[connIndex].SetReadDeadline(s.lastReadDeadline)
	}
	if !s.lastWriteDeadline.IsZero() {
		_ = s.conns[connIndex].SetWriteDeadline(s.lastWriteDeadline)
	}
	b := s.lastWrite
	s.replayMu.RUnlock()
	if b != nil {
		_, _ = s.conns[connIndex].Write(b)
	}
	close(s.replay[connIndex]) // fire off any pending iter's
}

func (s *staggerConn) getConnCount() int {
	count := int(s.connCount.Load())
	if count > len(s.conns) {
		return len(s.conns)
	}
	return count
}

func (s *staggerConn) iter(op connOp, fn func(conn PacketConn) (keepGoing bool, err error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, len(s.conns))
	errs := make(chan error, len(s.conns))
	for ix, conn := range s.conns {
		go func(ix int, conn PacketConn) {
			select {
			case <-s.replay[ix]:
			case <-ctx.Done():
				return
			}

			keepGoing, err := fn(conn)
			if err != nil {
				errs <- err
			} else if op == readOp || op == readFromOp {
				s.firstResponder.CAS(-1, int64(ix))
			}
			done <- struct{}{}
			if !keepGoing {
				cancel()
			}
		}(ix, conn)
	}
	for i := 0; i < s.getConnCount(); i++ {
		select {
		case <-done:
		case <-ctx.Done():
			goto ctxDone
		}
	}

ctxDone:
	if len(errs) >= s.getConnCount() {
		// only return an error if all conns failed
		err := <-errs
		return errors.Wrapf(err, "all connections have failed for %q", op)
	}
	return nil
}

func (s *staggerConn) Read(b []byte) (n int, err error) {
	type byteResp struct {
		n int
		b []byte
	}
	success := make(chan byteResp, len(s.conns))
	failure := make(chan byteResp, len(s.conns))
	err = s.iter(readOp, func(conn PacketConn) (bool, error) {
		buf := make([]byte, len(b))
		n, err := conn.Read(buf)
		if err == nil {
			success <- byteResp{n: n, b: buf}
		} else {
			failure <- byteResp{n: n, b: buf}
		}
		return err != nil, err
	})
	select {
	case resp := <-success:
		n = resp.n
		copy(b, resp.b)
	case resp := <-failure:
		n = resp.n
		copy(b, resp.b)
	}
	return
}

func (s *staggerConn) Write(b []byte) (n int, err error) {
	s.replayMu.Lock()
	s.lastWrite = b[:]
	s.replayMu.Unlock()
	success := make(chan int, len(s.conns))
	failure := make(chan int, len(s.conns))
	err = s.iter(writeOp, func(conn PacketConn) (bool, error) {
		n, err := conn.Write(b)
		if err == nil {
			success <- n
		} else {
			failure <- n
		}
		return true, err
	})
	select {
	case n = <-success:
	case n = <-failure:
	}
	return
}

// ReadFrom implements net.PacketConn, but is unused for UDP DNS queries. May require further testing for other use cases.
func (s *staggerConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	type byteResp struct {
		n    int
		addr net.Addr
		b    []byte
	}
	success := make(chan byteResp, len(s.conns))
	failure := make(chan byteResp, len(s.conns))
	err = s.iter(readFromOp, func(conn PacketConn) (bool, error) {
		buf := make([]byte, len(b))
		n, addr, err := conn.ReadFrom(buf)
		if err == nil {
			success <- byteResp{n: n, addr: addr, b: buf}
		} else {
			failure <- byteResp{n: n, addr: addr, b: buf}
		}
		return err != nil, err
	})
	select {
	case resp := <-success:
		n = resp.n
		addr = resp.addr
		copy(b, resp.b)
	case resp := <-failure:
		n = resp.n
		addr = resp.addr
		copy(b, resp.b)
	}
	return
}

// WriteTo implements net.PacketConn, but is unused for UDP DNS queries. May require further testing for other use cases.
func (s *staggerConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	s.replayMu.Lock()
	s.lastWrite = b[:]
	s.lastWriteAddr = addr
	s.replayMu.Unlock()
	success := make(chan int, len(s.conns))
	failure := make(chan int, len(s.conns))
	err = s.iter(writeToOp, func(conn PacketConn) (bool, error) {
		n, err := conn.WriteTo(b, addr)
		if err == nil {
			success <- n
		} else {
			failure <- n
		}
		return true, err
	})
	select {
	case n = <-success:
	case n = <-failure:
	}
	return
}

func (s *staggerConn) Close() error {
	s.replayMu.RLock()
	s.tickerCancel()
	s.replayMu.RUnlock()
	var foundErr error
	for _, conn := range s.conns {
		err := conn.Close()
		if err != nil {
			foundErr = err
		}
	}
	return foundErr
}

func (s *staggerConn) LocalAddr() net.Addr {
	// This doesn't seem to matter for DNS connections, and it is not clear what a more appropriate result should be.
	return s.conns[0].LocalAddr()
}

func (s *staggerConn) RemoteAddr() net.Addr {
	// This doesn't seem to matter for DNS connections, and it is not clear what a more appropriate result should be.
	return s.conns[0].RemoteAddr()
}

func (s *staggerConn) SetDeadline(t time.Time) error {
	s.replayMu.Lock()
	s.lastDeadline = t
	s.replayMu.Unlock()
	return s.iter(setDeadlineOp, func(conn PacketConn) (bool, error) {
		return true, conn.SetDeadline(t)
	})
}

func (s *staggerConn) SetReadDeadline(t time.Time) error {
	s.replayMu.Lock()
	s.lastReadDeadline = t
	s.replayMu.Unlock()
	return s.iter(setReadDeadlineOp, func(conn PacketConn) (bool, error) {
		return true, conn.SetReadDeadline(t)
	})
}

func (s *staggerConn) SetWriteDeadline(t time.Time) error {
	s.replayMu.Lock()
	s.lastWriteDeadline = t
	s.replayMu.Unlock()
	return s.iter(setWriteDeadlineOp, func(conn PacketConn) (bool, error) {
		return true, conn.SetWriteDeadline(t)
	})
}
