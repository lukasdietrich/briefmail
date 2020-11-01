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

package smtp

import (
	"strconv"
	"time"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

type sessionState uint

const (
	sInit sessionState = iota
	sHelo
	sMail
	sRcpt
)

func (s sessionState) String() string {
	return [...]string{
		"init",
		"helo",
		"mail",
		"rcpt",
	}[s]
}

func (s sessionState) in(any ...sessionState) bool {
	for _, other := range any {
		if other == s {
			return true
		}
	}

	return false
}

type session struct {
	textproto.Conn

	state    sessionState
	envelope delivery.Envelope
	headers  []hook.HeaderField
	mailbox  *models.MailboxEntity
}

func (s *session) isSubmission() bool {
	return s.mailbox != nil
}

func (s *session) reply(code int, text string) error {
	if err := s.SetWriteTimeout(time.Minute * 5); err != nil {
		return err
	}

	s.WriteString(strconv.Itoa(code))
	s.WriteString(" ")
	s.WriteString(text)
	s.Endline()

	return s.Flush()
}

func (s *session) read(c *command) error {
	if err := s.SetReadTimeout(time.Minute * 5); err != nil {
		return err
	}

	return c.readFrom(s)
}
