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
	"context"
	"crypto/tls"
	"io"
	"net"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

// Proto is a pop3 protocol implementation.
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
			"capa": capa(
				"USER",
				"UIDL"),

			"user": user(),
			"pass": pass(locks, authenticator, inboxer),

			"stat": stat(),
			"list": list(),
			"uidl": uidl(),
			"retr": retr(inboxer, blobs),
			"dele": dele(),

			"noop": noop(),
			"rset": rset(),
			"quit": quit(inboxer),

			"stls": stls(tlsConfig),
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

// Handle accepts a pop3 connection and handles all incoming commands in a loop until the
// transmission is closed.
func (p *Proto) Handle(c textproto.Conn) {
	s := &session{
		Conn:  c,
		state: sInit,
	}

	if err := s.send(&rReady); err != nil {
		return
	}

	ctx := log.WithOrigin(c.Context(), "pop3")
	log.InfoContext(ctx).Msg("starting session")

	switch err := p.loop(ctx, s); err {
	case io.EOF, errCloseSession, nil:
		log.InfoContext(ctx).Msg("session closed")
		s.send(&rBye)
	default:
		log.ErrorContext(ctx).
			Err(err).
			Msg("session closed with an error")

		if errt, ok := err.(*net.OpError); ok && errt.Timeout() {
			s.send(&rTimeout)
		} else {
			s.send(&rError)
		}
	}

	if s.state == sTransaction {
		log.InfoContext(ctx).
			Int64("mailbox", s.mailbox.ID).
			Msg("unlocking mailbox")

		p.locks.unlock(s.mailbox.ID)
	}
}

func (p *Proto) loop(ctx context.Context, s *session) error {
	var cmd command

	for {
		if err := s.read(&cmd); err != nil {
			return err
		}

		commandName := string(bytes.ToLower(cmd.head))
		ctx := log.WithCommand(ctx, commandName)
		h, ok := p.handlerMap[commandName]

		if !ok {
			log.DebugContext(ctx).Msg("command not implemented")

			if err := s.send(&rNotImplemented); err != nil {
				return err
			}

			continue
		}

		if err := h(ctx, s, &cmd); err != nil {
			if err != errCloseSession {
				log.DebugContext(ctx).
					Err(err).
					Msg("error during command")
			}

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
