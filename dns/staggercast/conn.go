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
}

// PacketConn implements both net.Conn and net.PacketConn
type PacketConn interface {
	io.Reader
	io.Writer
	RemoteAddr() net.Addr
	net.PacketConn
}

// staggerConn fires out all Writes to all outgoing connections, Reads return the first successful read.
type staggerConn struct {
	conns []PacketConn

	connCount    *atomic.Uint64
	replay       []chan struct{}
	replayMu     sync.RWMutex
	tickerCancel context.CancelFunc
	// capture last Write and SetDeadlines for replay on staggered connections
	lastDeadline, lastReadDeadline, lastWriteDeadline time.Time
	lastWrite                                         []byte
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
	}
}

// Stagger controls when secondary connections are attempted after the first. For any new enabled connection, the last Write and SetDeadlines are replayed onto it.
// If Stagger is called after the connection is started, then the behavior is undefined.
// First connection is attempted immediately, then all future connections will be enabled when receiving from ticker.
//
// Example: Create and wrap a 'time.Ticker.C' with a duration of 1 second.
// The first connection is attempted immediately. For every 1 second the Read hasn't returned, an additional connection is attempted.
// Once a connection succeeds, the result is returned immediately from Read.
func (p *staggerConn) Stagger(ticker <-chan struct{}, cancel context.CancelFunc) {
	p.replayMu.Lock()

	ctx, tickerCancel := context.WithCancel(context.Background())
	p.tickerCancel = tickerCancel

	totalLength := len(p.conns)
	p.replay = make([]chan struct{}, totalLength)
	for i := range p.replay {
		p.replay[i] = make(chan struct{})
	}
	close(p.replay[0]) // initial connection should immediately fire
	connCount := atomic.NewUint64(1)
	p.connCount = connCount

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
				go p.runReplay(count - 1)
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()
	p.replayMu.Unlock()
}

// runReplay synchronously reapplies the last Deadlines and the last Write
func (p *staggerConn) runReplay(connIndex uint64) {
	p.replayMu.RLock()
	if !p.lastDeadline.IsZero() {
		p.conns[connIndex].SetDeadline(p.lastDeadline)
	}
	if !p.lastReadDeadline.IsZero() {
		p.conns[connIndex].SetReadDeadline(p.lastReadDeadline)
	}
	if !p.lastWriteDeadline.IsZero() {
		p.conns[connIndex].SetWriteDeadline(p.lastWriteDeadline)
	}
	b := p.lastWrite
	p.replayMu.RUnlock()
	if b != nil {
		p.conns[connIndex].Write(b)
	}
	close(p.replay[connIndex]) // fire off any pending iter's
}

func (p *staggerConn) getConnCount() int {
	count := int(p.connCount.Load())
	if count > len(p.conns) {
		return len(p.conns)
	}
	return count
}

func (p *staggerConn) iter(op string, fn func(conn PacketConn) (keepGoing bool, err error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var started atomic.Int64
	done := make(chan struct{}, len(p.conns))
	errs := make(chan error, len(p.conns))
	for ix, conn := range p.conns {
		go func(ix int, conn PacketConn) {
			select {
			case <-p.replay[ix]:
			case <-ctx.Done():
				return
			}

			started.Inc()
			keepGoing, err := fn(conn)
			if err != nil {
				errs <- err
			}
			done <- struct{}{}
			if !keepGoing {
				cancel()
			}
		}(ix, conn)
	}
	for i := 0; i < p.getConnCount(); i++ {
		select {
		case <-done:
		case <-ctx.Done():
			goto ctxDone
		}
	}

ctxDone:
	if len(errs) >= p.getConnCount() {
		// only return an error if all conns failed
		err := <-errs
		return errors.Wrapf(err, "all connections have failed for %q", op)
	}
	return nil
}

func (p *staggerConn) Read(b []byte) (n int, err error) {
	type byteResp struct {
		n int
		b []byte
	}
	success := make(chan byteResp, len(p.conns))
	failure := make(chan byteResp, len(p.conns))
	err = p.iter("read", func(conn PacketConn) (bool, error) {
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

func (p *staggerConn) Write(b []byte) (n int, err error) {
	p.replayMu.Lock()
	p.lastWrite = b[:]
	p.replayMu.Unlock()
	success := make(chan int, len(p.conns))
	failure := make(chan int, len(p.conns))
	err = p.iter("write", func(conn PacketConn) (bool, error) {
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
func (p *staggerConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	panic("read from")
	type readResponse struct {
		n    int
		addr net.Addr
		b    []byte
	}
	success := make(chan readResponse, len(p.conns))
	failure := make(chan readResponse, len(p.conns))
	err = p.iter("read from", func(conn PacketConn) (bool, error) {
		buf := make([]byte, len(b))
		n, addr, err := conn.ReadFrom(buf)
		if err == nil {
			success <- readResponse{n: n, addr: addr, b: buf}
		} else {
			failure <- readResponse{n: n, addr: addr, b: buf}
		}
		return err != nil, err
	})
	select {
	case resp := <-success:
		n, addr = resp.n, resp.addr
		copy(b, resp.b)
	case resp := <-failure:
		n, addr = resp.n, resp.addr
		copy(b, resp.b)
	}
	return
}

// WriteTo implements net.PacketConn, but is unused for UDP DNS queries. May require further testing for other use cases.
func (p *staggerConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	panic("write to")
	p.replayMu.Lock()
	p.lastWrite = b[:]
	p.replayMu.Unlock()
	success := make(chan int, len(p.conns))
	failure := make(chan int, len(p.conns))
	err = p.iter("write to", func(conn PacketConn) (bool, error) {
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

func (p *staggerConn) Close() error {
	p.replayMu.RLock()
	p.tickerCancel()
	p.replayMu.RUnlock()
	// TODO Can this be done with iter?
	var foundErr error
	for _, conn := range p.conns {
		err := conn.Close()
		if err != nil {
			foundErr = err
		}
	}
	return foundErr
}

func (p *staggerConn) LocalAddr() net.Addr {
	// This doesn't seem to matter for DNS connections, and it is not clear what a more appropriate result should be.
	return p.conns[0].LocalAddr()
}

func (p *staggerConn) RemoteAddr() net.Addr {
	// This doesn't seem to matter for DNS connections, and it is not clear what a more appropriate result should be.
	return p.conns[0].RemoteAddr()
}

func (p *staggerConn) SetDeadline(t time.Time) error {
	p.replayMu.Lock()
	p.lastDeadline = t
	p.replayMu.Unlock()
	return p.iter("deadline", func(conn PacketConn) (bool, error) {
		return true, conn.SetDeadline(t)
	})
}

func (p *staggerConn) SetReadDeadline(t time.Time) error {
	p.replayMu.Lock()
	p.lastReadDeadline = t
	p.replayMu.Unlock()
	return p.iter("read deadline", func(conn PacketConn) (bool, error) {
		return true, conn.SetReadDeadline(t)
	})
}

func (p *staggerConn) SetWriteDeadline(t time.Time) error {
	p.replayMu.Lock()
	p.lastWriteDeadline = t
	p.replayMu.Unlock()
	return p.iter("write deadline", func(conn PacketConn) (bool, error) {
		return true, conn.SetWriteDeadline(t)
	})
}
