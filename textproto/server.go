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

import "net"

// Server is a general purpose tcp server for text based protocols like SMTP
// or POP3.
type Server interface {
	// Listen will open a new tcp listener and block until an error occurs.
	// An error is either returned when trying to bind the given address or
	// whenever accepting a new connection fails.
	Listen(addr string) error
}

// Protocol is an interface for text based protocol implementations.
type Protocol interface {
	// Handle is supposed to consume a connection and manage all traffic
	// over it. Once Handle returns, the underlying network connection is
	// automatically closed by the server.
	Handle(Conn)
}

type server struct {
	proto Protocol
}

// NewServer returns a Server using a specified protocol implementation.
// The Server has to be started explitly afterwards.
func NewServer(proto Protocol) Server {
	return &server{
		proto: proto,
	}
}

func (s *server) Listen(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go s.handle(conn)
	}
}

func (s *server) handle(conn net.Conn) {
	defer conn.Close() // nolint:errcheck

	s.proto.Handle(wrapConn(conn))
}
