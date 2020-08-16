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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var (
	errCloseSession  = errors.New("pop3: session closed")
	errBadSequence   = errors.New("pop3: bad sequence of commands")
	errInvalidSyntax = errors.New("pop3: invalid syntax")
)

type handler func(*session, *command) error

// `USER` command as specified in RFC#1939
//
//     "USER" <mailbox> CRLF
func user() handler {
	rOk := reply{true, "now the secret"}

	return func(s *session, c *command) error {
		if !s.state.in(sInit, sUser) {
			return errBadSequence
		}

		args := c.args()

		if len(args) != 1 {
			return errInvalidSyntax
		}

		s.name = string(args[0])
		s.state = sUser

		return s.send(&rOk)
	}
}

// `PASS` command as specified in RFC#1939
//
//     "PASS" <password> CRLF
func pass(l *locks, authenticator *delivery.Authenticator, inboxer *delivery.Inboxer) handler {
	var (
		rOk        = reply{true, "I knew it was you!"}
		rWrongPass = reply{false, "nice try"}
		rLocked    = reply{false, "there is two of you?"}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sUser) {
			return errBadSequence
		}

		args := c.args()

		if len(args) != 1 {
			return errInvalidSyntax
		}

		addr, err := mails.ParseUnicode(s.name)
		if err != nil {
			return err
		}

		mailbox, err := authenticator.Auth(s.Context(), addr, args[0])
		if err != nil {
			if errors.Is(err, delivery.ErrWrongAddressPassword) {
				return s.send(&rWrongPass)
			}

			return err
		}

		if !l.lock(mailbox.ID) {
			return s.send(&rLocked)
		}

		s.inbox, err = inboxer.Inbox(s.Context(), mailbox)
		if err != nil {
			return err
		}

		s.state = sTransaction

		return s.send(&rOk)
	}
}

// `QUIT` command as specified in RFC#1939
//
//     "QUIT" CRLF
func quit(inboxer *delivery.Inboxer) handler {
	return func(s *session, _ *command) error {
		if s.state.in(sTransaction) {
			if err := inboxer.Commit(s.Context(), s.mailbox, s.inbox); err != nil {
				return err
			}
		}

		return errCloseSession
	}
}

// `STAT` command as specified in RFC#1939
//
//     "STAT" CRLF
func stat() handler {
	return func(s *session, _ *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		return s.send(&reply{
			true,
			fmt.Sprintf("%d %d", s.inbox.Count(), s.inbox.Size()),
		})
	}
}

// `LIST` command as specified in RFC#1939
//
//     "LIST" [ id ] CRLF
func list() handler {
	rNoMessage := reply{false, "no such message"}

	return func(s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		args := c.args()

		switch len(args) {
		case 0:
			s.send(&reply{
				true,
				fmt.Sprintf("%d messages (%d octets)", s.inbox.Count(), s.inbox.Size()),
			})

			for i, mail := range s.inbox.Mails {
				if s.inbox.IsMarked(i) {
					continue
				}

				fmt.Fprintf(s, "%d %d", i+1, mail.Size)
				s.Endline()
			}

			s.WriteString(".")
			s.Endline()

			return s.Flush()

		case 1:
			n, err := strconv.ParseInt(string(args[0]), 10, 64)
			if err != nil {
				return errInvalidSyntax
			}

			index := int(n - 1)

			if index < 0 || index >= len(s.inbox.Mails) || s.inbox.IsMarked(index) {
				return s.send(&rNoMessage)
			}

			return s.send(&reply{
				true,
				fmt.Sprintf("%d %d", n, s.inbox.Mails[index].Size),
			})

		default:
			return errInvalidSyntax
		}
	}
}

// `UIDL` command as specified in RFC#1939
//
//     "UIDL" [ id ] CRLF
func uidl() handler {
	var (
		rOk        = reply{true, "list follows"}
		rNoMessage = reply{false, "no such message"}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		args := c.args()

		switch len(args) {
		case 0:
			s.send(&rOk)

			for i, mail := range s.inbox.Mails {
				if s.inbox.IsMarked(i) {
					continue
				}

				fmt.Fprintf(s, "%d %s", i+1, mail.ID)
				s.Endline()
			}

			s.WriteString(".")
			s.Endline()

			return s.Flush()

		case 1:
			n, err := strconv.ParseInt(string(args[0]), 10, 64)
			if err != nil {
				return errInvalidSyntax
			}

			index := int(n - 1)

			if index < 0 || index >= len(s.inbox.Mails) || s.inbox.IsMarked(index) {
				return s.send(&rNoMessage)
			}

			return s.send(&reply{
				true,
				fmt.Sprintf("%d %s", n, s.inbox.Mails[index].ID),
			})

		default:
			return errInvalidSyntax
		}
	}
}

// `RETR` command as specified in RFC#1939
//
//     "RETR" <id> CRLF
func retr(inboxer *delivery.Inboxer, blobs *storage.Blobs) handler {
	var (
		rOk        = reply{true, "message coming"}
		rNoMessage = reply{false, "no such message"}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		args := c.args()

		if len(args) != 1 {
			return errInvalidSyntax
		}

		n, err := strconv.ParseInt(string(args[0]), 10, 64)
		if err != nil {
			return errInvalidSyntax
		}

		index := int(n - 1)

		if index < 0 || index >= len(s.inbox.Mails) || s.inbox.IsMarked(index) {
			return s.send(&rNoMessage)
		}

		if err := s.send(&rOk); err != nil {
			return err
		}

		r, err := blobs.Reader(s.inbox.Mails[index].ID)
		if err != nil {
			return err
		}

		w := s.DotWriter()

		_, err = io.Copy(w, r)

		r.Close()
		w.Close()
		s.Flush()

		return err
	}
}

// `DELE` command as specified in RFC#1939
//
//     "DELE" <id> CRLF
func dele() handler {
	var (
		rOk        = reply{true, "woosh"}
		rNoMessage = reply{false, "no such message"}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		args := c.args()

		if len(args) != 1 {
			return errInvalidSyntax
		}

		n, err := strconv.ParseInt(string(args[0]), 10, 64)
		if err != nil {
			return errInvalidSyntax
		}

		index := int(n - 1)

		if index < 0 || index >= len(s.inbox.Mails) || s.inbox.IsMarked(index) {
			return s.send(&rNoMessage)
		}

		s.inbox.Mark(index)
		return s.send(&rOk)
	}
}

// `NOOP` command as specified in RFC#1939
//
//     "NOOP" CRLF
func noop() handler {
	rOk := reply{true, "what did you expect?"}

	return func(s *session, _ *command) error {
		return s.send(&rOk)
	}
}

// `RSET` command as specified in RFC#1939
//
//     "RSET" CRLF
func rset() handler {
	rOk := reply{true, "lost some intel during time travel"}

	return func(s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		s.inbox.Reset()
		return s.send(&rOk)
	}
}

// `STLS` command as specified in RFC#2595
//
//     "STLS" CRLF
func stls(config *tls.Config) handler {
	var (
		rReady          = reply{true, "ready to go undercover."}
		rTLSUnavailable = reply{false, "I am afraid, I lost my disguise!"}
		rAlreadyTLS     = reply{false, "what are you afraid of?"}
	)

	return func(s *session, _ *command) error {
		if config == nil {
			return s.send(&rTLSUnavailable)
		}

		if s.IsTLS() {
			return s.send(&rAlreadyTLS)
		}

		if err := s.send(&rReady); err != nil {
			return err
		}

		return s.UpgradeTLS(config)
	}
}

// `CAPA` command as specified in RFC#2449
//
//     "CAPA" CRLF
func capa(capabilities ...string) handler {
	var (
		rOk = reply{true, "I can do some things"}
	)

	return func(s *session, _ *command) error {
		if err := s.send(&rOk); err != nil {
			return err
		}

		for _, capability := range capabilities {
			s.WriteString(capability)
			s.Endline()
		}

		s.WriteString(".")
		s.Endline()

		return s.Flush()
	}
}
