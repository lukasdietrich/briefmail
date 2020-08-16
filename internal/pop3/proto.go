// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
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

package pop3

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

var log = logrus.WithField("prefix", "pop3")

type Proto struct {
	locks      *locks
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(
	authenticator *delivery.Authenticator,
	inboxer *delivery.Inboxer,
	blobs *storage.Blobs,
	tlsConfig *tls.Config,
) *Proto {
	locks := newLocks()

	return &Proto{
		locks: locks,
		handlerMap: map[string]handler{
			"CAPA": capa(
				"USER",
				"UIDL"),

			"USER": user(),
			"PASS": pass(locks, authenticator, inboxer),

			"STAT": stat(),
			"LIST": list(),
			"UIDL": uidl(),
			"RETR": retr(inboxer, blobs),
			"DELE": dele(),

			"NOOP": noop(),
			"RSET": rset(),
			"QUIT": quit(inboxer),

			"STLS": stls(tlsConfig),
		},
	}
}

var (
	rReady          = reply{true, "ready"}
	rBye            = reply{true, "closing transmission channel"}
	rTimeout        = reply{false, "timed out"}
	rError          = reply{false, "action aborted: local error in processing"}
	rNotImplemented = reply{false, "command not implemented"}
	rBadSequence    = reply{false, "bad sequence of commands"}
	rInvalidSyntax  = reply{false, "invalid syntax"}
)

func (p *Proto) Handle(c textproto.Conn) {
	s := &session{
		Conn:  c,
		state: sInit,
	}

	if err := s.send(&rReady); err != nil {
		return
	}

	switch err := p.loop(s); err {
	case io.EOF, errCloseSession, nil:
		s.send(&rBye)
	default:
		log.Warn(err)

		if errt, ok := err.(*net.OpError); ok && errt.Timeout() {
			s.send(&rTimeout)
		} else {
			s.send(&rError)
		}
	}

	if s.state == sTransaction {
		p.locks.unlock(s.mailbox.ID)
	}
}

func (p *Proto) loop(s *session) error {
	var cmd command

	for {
		if err := s.read(&cmd); err != nil {
			return err
		}

		h, ok := p.handlerMap[string(bytes.ToUpper(cmd.head))]

		if !ok {
			if err := s.send(&rNotImplemented); err != nil {
				return err
			}

			continue
		}

		if err := h(s, &cmd); err != nil {
			switch err {
			case errBadSequence:
				if err := s.send(&rBadSequence); err != nil {
					return err
				}

			case errInvalidSyntax:
				if err := s.send(&rInvalidSyntax); err != nil {
					return err
				}

			default:
				return err
			}
		}
	}
}
