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

	"github.com/sirupsen/logrus"

	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/storage"
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
func pass(l *locks, db *storage.DB) handler {
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

		id, ok, err := db.Authenticate(s.name, string(args[0]))
		if err != nil {
			return err
		}

		if !ok {
			return s.send(&rWrongPass)
		}

		if !l.lock(id) {
			return s.send(&rLocked)
		}

		s.mailbox.entries, s.mailbox.size, err = db.Entries(id)
		if err != nil {
			return err
		}

		s.mailbox.id = id
		s.mailbox.marks = make(map[int64]bool)

		s.state = sTransaction

		return s.send(&rOk)
	}
}

// `QUIT` command as specified in RFC#1939
//
//     "QUIT" CRLF
func quit(db *storage.DB, blobs *storage.Blobs) handler {
	return func(s *session, _ *command) error {
		if s.state.in(sTransaction) {
			log.WithFields(logrus.Fields{
				"mailbox": s.mailbox.id,
				"tls":     s.Conn.IsTLS(),
			}).Debugf("end of transaction (delete %d of %d)",
				len(s.mailbox.marks),
				len(s.mailbox.entries))

			mails := make([]model.ID, 0, len(s.mailbox.marks))

			for n := range s.mailbox.marks {
				mails = append(mails, s.mailbox.entries[int(n)].MailID)
			}

			if err := db.DeleteEntries(mails, s.mailbox.id); err != nil {
				return err
			}

			orphans, err := db.DeleteOrphans()
			if err != nil {
				return err
			}

			for _, orphan := range orphans {
				blobs.Delete(orphan)
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
			fmt.Sprintf("%d %d", len(s.mailbox.entries), s.mailbox.size),
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
				fmt.Sprintf("%d messages (%d octets)",
					len(s.mailbox.entries)-len(s.mailbox.marks),
					s.mailbox.size-s.mailbox.sizeDel),
			})

			for i, entry := range s.mailbox.entries {
				if s.mailbox.marks[int64(i)] {
					continue
				}

				fmt.Fprintf(s, "%d %d", i+1, entry.Size)
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

			n--

			if n < 0 || n >= int64(len(s.mailbox.entries)) || s.mailbox.marks[n] {
				return s.send(&rNoMessage)
			}

			return s.send(&reply{
				true,
				fmt.Sprintf("%d %d", n, s.mailbox.entries[n].Size),
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

			for i, entry := range s.mailbox.entries {
				if s.mailbox.marks[int64(i)] {
					continue
				}

				fmt.Fprintf(s, "%d %s", i+1, entry.MailID)
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

			n--

			if n < 0 || n >= int64(len(s.mailbox.entries)) || s.mailbox.marks[n] {
				return s.send(&rNoMessage)
			}

			return s.send(&reply{
				true,
				fmt.Sprintf("%d %s", n, s.mailbox.entries[n].MailID),
			})

		default:
			return errInvalidSyntax
		}
	}
}

// `RETR` command as specified in RFC#1939
//
//     "RETR" <id> CRLF
func retr(blobs *storage.Blobs) handler {
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

		n--

		if n < 0 || n >= int64(len(s.mailbox.entries)) || s.mailbox.marks[n] {
			return s.send(&rNoMessage)
		}

		if err := s.send(&rOk); err != nil {
			return err
		}

		r, err := blobs.Read(s.mailbox.entries[n].MailID)
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

		n--

		if n < 0 || n >= int64(len(s.mailbox.entries)) || s.mailbox.marks[n] {
			return s.send(&rNoMessage)
		}

		s.mailbox.marks[n] = true
		s.mailbox.sizeDel += s.mailbox.entries[n].Size

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

		if len(s.mailbox.marks) > 0 {
			s.mailbox.marks = make(map[int64]bool)
			s.mailbox.sizeDel = 0
		}

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
			s.WriteString(capability) // nolint:errcheck
			s.Endline()               // nolint:errcheck
		}

		s.WriteString(".")
		s.Endline()

		return s.Flush()
	}
}
