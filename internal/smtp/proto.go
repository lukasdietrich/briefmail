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

package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

// Proto is a smtp server protocol implementation.
type Proto struct {
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(
	authenticator *delivery.Authenticator,
	mailman *delivery.Mailman,
	addressbook *delivery.Addressbook,
	cache *storage.Cache,
	tlsConfig *tls.Config,
	fromHooks []hook.FromHook,
	dataHooks []hook.DataHook,
) *Proto {
	var (
		hostname = viper.GetString("general.hostname")
		maxSize  = viper.GetInt64("mail.size")
	)

	return &Proto{
		handlerMap: map[string]handler{
			"helo": helo(hostname),
			"ehlo": ehlo(hostname,
				fmt.Sprintf("SIZE %d", maxSize),
				fmt.Sprintf("STARTTLS"),
				fmt.Sprintf("AUTH %s %s", "PLAIN", "LOGIN"),
			),

			"mail": mail(addressbook, maxSize, fromHooks),
			"rcpt": rcpt(addressbook),
			"data": data(mailman, cache, maxSize, dataHooks),

			"noop": noop(),
			"rset": rset(),
			"vrfy": vrfy(),
			"quit": quit(),

			"starttls": starttls(tlsConfig),
			"auth":     auth(authenticator),
		},
	}
}

var (
	rReady          = reply{220, "ready"}
	rBye            = reply{221, "closing transmission channel"}
	rError          = reply{451, "action aborted: local error in processing"}
	rPathTooLong    = reply{501, "path too long"}
	rCommandSyntax  = reply{501, "syntax error in parameters or arguments"}
	rNotImplemented = reply{502, "command not implemented"}
	rBadSequence    = reply{503, "bad sequence of commands"}
	rInvalidAddress = reply{553, "invalid address format"}
)

// Handle accepts an smtp connection and handles all incoming commands in a loop until the
// transmission is closed.
func (p *Proto) Handle(c textproto.Conn) {
	s := &session{
		Conn:  c,
		state: sInit,
		envelope: mails.Envelope{
			Addr: c.RemoteAddr(),
		},
	}

	if err := s.send(&rReady); err != nil {
		return
	}

	ctx := log.WithOrigin(c.Context(), "smtp")
	log.InfoContext(ctx).Msg("starting session")

	switch err := p.loop(ctx, s); err {
	case io.EOF, errCloseSession, nil:
		log.InfoContext(ctx).Msg("session closed")
		s.send(&rBye)
	default:
		log.ErrorContext(ctx).
			Err(err).
			Msg("session closed with an error")

		s.send(&rError)
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

			case errCommandSyntax:
				if err := s.send(&rCommandSyntax); err != nil {
					return err
				}

			case mails.ErrInvalidAddressFormat:
				if err := s.send(&rInvalidAddress); err != nil {
					return err
				}

			case mails.ErrPathTooLong:
				if err := s.send(&rPathTooLong); err != nil {
					return err
				}

			default:
				return err
			}
		}
	}
}
