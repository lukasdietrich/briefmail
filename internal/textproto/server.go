// Copyright (C) 2018  Lukas Dietrich <lukas@lukasdietrich.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package textproto

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
)

var (
	// ErrServerClosed is returned when a server is shut down.
	ErrServerClosed = errors.New("textproto: server closed")
)

// Server is a general purpose tcp server for text based protocols like SMTP
// or POP3.
type Server interface {
	// Listen will open a new tcp listener and block until an error occurs.
	// An error is either returned when trying to bind the given address or
	// whenever accepting a new connection fails.
	Listen(addr string) error

	// Shutdown gracefully shuts down the Server. Repeated calls are not supported
	// and will result in a panic.
	// Once called, no more connections will be established. Pending connections
	// are still waited on until the context is canceled.
	Shutdown(ctx context.Context)
}

// Protocol is an interface for text based protocol implementations.
type Protocol interface {
	// Handle is supposed to consume a connection and manage all traffic
	// over it. Once Handle returns, the underlying network connection is
	// automatically closed by the server.
	Handle(Conn)
}

type server struct {
	proto     Protocol
	tlsConfig *tls.Config
	l         net.Listener
	wg        sync.WaitGroup
	conns     map[*conn]struct{}
	closing   chan struct{}
}

// NewServer returns a Server using a specified protocol implementation.
// If the provided *tls.Config is non-nil, the Server will accept only
// connections over tls.
// The Server has to be started explicitly afterwards.
func NewServer(proto Protocol, tlsConfig *tls.Config) Server {
	return &server{
		proto:     proto,
		tlsConfig: tlsConfig,
		conns:     make(map[*conn]struct{}),
		closing:   make(chan struct{}),
	}
}

func (s *server) Shutdown(ctx context.Context) {
	close(s.closing)
	s.l.Close()

	ctx, cancelFunc := context.WithCancel(ctx)
	go s.waitForClients(cancelFunc)

	<-ctx.Done()

	for conn := range s.conns {
		conn.raw.Close()
	}
}

// waitForClients waits for all pending clients to close, before calling the
// callback function.
func (s *server) waitForClients(callback func()) {
	s.wg.Wait()
	callback()
}

func (s *server) Listen(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.l = l

	for {
		select {
		case <-s.closing:
			return ErrServerClosed

		default:
			if err := s.acceptConn(); err != nil {
				return err
			}
		}
	}
}

func (s *server) acceptConn() error {
	conn, err := s.l.Accept()
	if err != nil {
		select {
		case <-s.closing:
			return ErrServerClosed

		default:
			return err
		}
	}

	s.wg.Add(1)

	go s.handle(wrapConn(conn))
	return nil
}

func (s *server) handle(c *conn) {
	defer s.wg.Done()
	defer delete(s.conns, c)
	defer c.raw.Close()

	s.conns[c] = struct{}{}

	if s.tlsConfig != nil {
		if c.UpgradeTLS(s.tlsConfig) != nil {
			return
		}
	}

	s.proto.Handle(c)
}
