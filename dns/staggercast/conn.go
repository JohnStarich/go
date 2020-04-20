package staggercast

// Staggercast implements a one-to-many net.Conn for easy scattering of the same request to multiple endpoints with control on when new connections are attempted.
// For ease of use in a DNS resolver Dial func, all connections implement both net.Conn and net.PacketConn.

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

// Conn implements both net.Conn and net.PacketConn
type Conn interface {
	io.Reader
	io.Writer
	RemoteAddr() net.Addr
	net.PacketConn
}

// staggerConn fires out all Writes to all outgoing connections, Reads return the first successful read.
type staggerConn struct {
	conns []Conn
	//ticker      func() (<-chan struct{}, context.CancelFunc)
}

func New(conns []Conn) Conn {
	if len(conns) == 0 {
		panic("connection count must be non-zero")
	}
	return &staggerConn{
		conns: conns,
	}
}

//TODO
//func (p *staggerConn) Stagger(tickerFn func() (<-chan struct{}, context.CancelFunc)) {
//p.ticker = tickerFn
//}

func (p *staggerConn) iter(op string, fn func(conn Conn) (keepGoing bool, err error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, len(p.conns))
	errs := make(chan error, len(p.conns))
	for _, conn := range p.conns {
		select {
		case <-ctx.Done():
			goto ctxDone // don't run on more connections if stopped early
		default:
		}

		go func(conn Conn) {
			keepGoing, err := fn(conn)
			if err != nil {
				errs <- err
			}
			done <- struct{}{}
			if !keepGoing {
				cancel()
			}
		}(conn)
	}
	for i := 0; i < len(p.conns); i++ {
		select {
		case <-done:
		case <-ctx.Done():
			goto ctxDone
		}
	}

ctxDone:
	if len(errs) == len(p.conns) {
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
	err = p.iter("read", func(conn Conn) (bool, error) {
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
	success := make(chan int, len(p.conns))
	failure := make(chan int, len(p.conns))
	err = p.iter("write", func(conn Conn) (bool, error) {
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
	type readResponse struct {
		n    int
		addr net.Addr
		b    []byte
	}
	success := make(chan readResponse, len(p.conns))
	failure := make(chan readResponse, len(p.conns))
	err = p.iter("read from", func(conn Conn) (bool, error) {
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
	success := make(chan int, len(p.conns))
	failure := make(chan int, len(p.conns))
	err = p.iter("write to", func(conn Conn) (bool, error) {
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
	return p.iter("close", func(conn Conn) (bool, error) {
		return true, conn.Close()
	})
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
	return p.iter("deadline", func(conn Conn) (bool, error) {
		return true, conn.SetDeadline(t)
	})
}

func (p *staggerConn) SetReadDeadline(t time.Time) error {
	return p.iter("read deadline", func(conn Conn) (bool, error) {
		return true, conn.SetReadDeadline(t)
	})
}

func (p *staggerConn) SetWriteDeadline(t time.Time) error {
	return p.iter("write deadline", func(conn Conn) (bool, error) {
		return true, conn.SetWriteDeadline(t)
	})
}
