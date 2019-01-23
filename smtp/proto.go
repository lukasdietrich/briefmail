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

	"github.com/lukasdietrich/briefmail/delivery"
	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/smtp/hook"
	"github.com/lukasdietrich/briefmail/storage"
	"github.com/lukasdietrich/briefmail/textproto"
)

// Config contains options for the SMTP protocol
type Config struct {
	Hostname string
	MaxSize  int64
	Mailman  *delivery.Mailman
	Cache    *storage.Cache
	TLS      *tls.Config
}

type proto struct {
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(config *Config) textproto.Protocol {
	return &proto{
		handlerMap: map[string]handler{
			"HELO": helo(config.Hostname),
			"EHLO": ehlo(config.Hostname,
				fmt.Sprintf("SIZE %d", config.MaxSize),
				fmt.Sprintf("STARTTLS"),
			),

			"MAIL": mail(config.MaxSize,
				hook.CheckSPF(),
			),
			"RCPT": rcpt(config.Mailman),
			"DATA": data(config.Mailman, config.Cache, config.MaxSize),

			"NOOP": noop(),
			"RSET": rset(),
			"VRFY": vrfy(),
			"QUIT": quit(),

			"STARTTLS": starttls(config.TLS),
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

func (p *proto) Handle(c textproto.Conn) {
	s := &session{
		Conn:  c,
		state: sInit,
		envelope: model.Envelope{
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
		logrus.Warn(err)
		s.send(&rError) // nolint:errcheck
	}
}

func (p *proto) loop(s *session) error {
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

			case model.ErrInvalidAddressFormat:
				if err := s.send(&rInvalidAddress); err != nil {
					return err
				}

			case model.ErrPathTooLong:
				if err := s.send(&rPathTooLong); err != nil {
					return err
				}

			default:
				return err
			}
		}
	}
}
