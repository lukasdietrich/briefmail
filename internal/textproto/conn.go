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
	"crypto/tls"
	"errors"
	"net"
	"time"
)

var (
	// ErrAlreadyTLS is returned if the connection already performed a tls
	// handshake.
	ErrAlreadyTLS = errors.New("textproto: already secured by tls")
)

// Conn is a wrapper around a network connection to enable line based reading
// and buffered writing.
type Conn interface {
	Reader
	Writer

	// SetReadTimeout sets the deadline for read calls to a time now + x.
	SetReadTimeout(time.Duration) error

	// SetWriteTimeout sets the deadline for write calls to a time now + x.
	SetWriteTimeout(time.Duration) error

	// UpgradeTLS replaces the underlying network connection with a tls
	// connection. Nothing happens, when an error occurred.
	UpgradeTLS(*tls.Config) error

	// IsTLS returns whether or not the connection is secured with tls.
	IsTLS() bool

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.IP
}

type variableNetConn struct {
	net.Conn
}

type conn struct {
	raw   *variableNetConn
	isTLS bool

	Reader
	Writer
}

func wrapConn(raw net.Conn) *conn {
	varConn := variableNetConn{Conn: raw}

	return &conn{
		raw: &varConn,

		Reader: newReader(&varConn),
		Writer: newWriter(&varConn),
	}
}

func (c *conn) SetReadTimeout(d time.Duration) error {
	return c.raw.SetReadDeadline(time.Now().Add(d))
}

func (c *conn) SetWriteTimeout(d time.Duration) error {
	return c.raw.SetWriteDeadline(time.Now().Add(d))
}

func (c *conn) UpgradeTLS(config *tls.Config) error {
	if c.IsTLS() {
		return ErrAlreadyTLS
	}

	tlsConn := tls.Server(c.raw.Conn, config)

	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	c.raw.Conn = tlsConn
	c.isTLS = true

	return nil
}

func (c *conn) IsTLS() bool {
	return c.isTLS
}

func (c *conn) RemoteAddr() net.IP {
	switch addr := c.raw.RemoteAddr().(type) {
	case *net.TCPAddr:
		return addr.IP

	case *net.UDPAddr:
		return addr.IP

	default:
		if addr.String() == "pipe" {
			// treat net/Pipe clients as localhost
			// for testing purposes
			return net.ParseIP("127.0.0.1")
		}
	}

	return nil
}
