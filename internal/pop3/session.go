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
	"time"

	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

type sessionState uint

const (
	sInit sessionState = iota
	sUser
	sTransaction
)

func (s sessionState) String() string {
	return [...]string{
		"init",
		"user",
		"transaction",
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

	state sessionState
	name  string

	mailbox struct {
		id      int64
		marks   map[int64]bool
		entries []storage.Entry
		size    int64
		sizeDel int64
	}
}

func (s *session) send(r *reply) error {
	if err := s.SetWriteTimeout(time.Minute * 5); err != nil {
		return err
	}

	return r.writeTo(s)
}

func (s *session) read(c *command) error {
	if err := s.SetReadTimeout(time.Minute * 5); err != nil {
		return err
	}

	return c.readFrom(s)
}
