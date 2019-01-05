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
	"io"

	"github.com/sirupsen/logrus"

	"github.com/lukasdietrich/briefmail/delivery"
	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/textproto"
)

// Config contains options for the SMTP protocol
type Config struct {
	Hostname string
	Mailman  *delivery.Mailman
}

type proto struct {
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(config *Config) textproto.Protocol {
	return &proto{
		handlerMap: map[string]handler{
			"HELO": helo(config.Hostname),
			"EHLO": ehlo(config.Hostname),

			"MAIL": mail(),
			"RCPT": rcpt(config.Mailman),
			"DATA": data(config.Mailman),

			"NOOP": noop(),
			"RSET": rset(),
			"VRFY": vrfy(),
			"QUIT": quit(),
		},
	}
}

var (
	rReady          = reply{220, "ready"}
	rBye            = reply{221, "closing transmission channel"}
	rError          = reply{451, "action aborted: local error in processing"}
	rPathTooLong    = reply{501, "path too long"}
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
		s.send(&rBye)
	default:
		logrus.Warn(err)
		s.send(&rError)
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
			s.send(&rNotImplemented)
			continue
		}

		if err := h(s, &cmd); err != nil {
			switch err {
			case errBadSequence:
				s.send(&rBadSequence)
			case model.ErrInvalidAddressFormat:
				s.send(&rInvalidAddress)
			case model.ErrPathTooLong:
				s.send(&rPathTooLong)
			default:
				return err
			}
		}
	}
}
