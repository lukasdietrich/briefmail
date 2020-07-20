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
	"crypto/tls"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/addressbook"
	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

var log = logrus.WithField("prefix", "smtp")

type Proto struct {
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(
	mailman delivery.Mailman,
	addressbook addressbook.Addressbook,
	cache *storage.Cache,
	db *storage.DB,
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
			"HELO": helo(hostname),
			"EHLO": ehlo(hostname,
				fmt.Sprintf("SIZE %d", maxSize),
				fmt.Sprintf("STARTTLS"),
				fmt.Sprintf("AUTH %s %s", "PLAIN", "LOGIN"),
			),

			"MAIL": mail(addressbook, maxSize, fromHooks),
			"RCPT": rcpt(mailman, addressbook),
			"DATA": data(mailman, cache, maxSize, dataHooks),

			"NOOP": noop(),
			"RSET": rset(),
			"VRFY": vrfy(),
			"QUIT": quit(),

			"STARTTLS": starttls(tlsConfig),
			"AUTH":     auth(db),
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

	switch err := p.loop(s); err {
	case io.EOF, errCloseSession, nil:
		s.send(&rBye) // nolint:errcheck
	default:
		log.Warn(err)
		s.send(&rError) // nolint:errcheck
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
