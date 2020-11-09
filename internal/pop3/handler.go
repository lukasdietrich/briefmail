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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var (
	errCloseSession  = errors.New("pop3: session closed")
	errBadSequence   = errors.New("pop3: bad sequence of commands")
	errInvalidSyntax = errors.New("pop3: invalid syntax")
)

type handler func(context.Context, *session, *command) error

// `USER` command as specified in RFC#1939
//
//     "USER" <mailbox> CRLF
func user() handler {
	return func(_ context.Context, s *session, c *command) error {
		if !s.state.in(sInit, sUser) {
			return errBadSequence
		}

		if len(c.args) != 1 {
			return errInvalidSyntax
		}

		s.name = c.args[0]
		s.state = sUser

		return s.reply(true, "now the secret")
	}
}

// `PASS` command as specified in RFC#1939
//
//     "PASS" <password> CRLF
func pass(l *locks, authenticator delivery.Authenticator, inboxer *delivery.Inboxer) handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sUser) {
			return errBadSequence
		}

		if len(c.args) != 1 {
			return errInvalidSyntax
		}

		mailbox, err := authenticator.Auth(ctx, s.name, c.args[0])
		if err != nil {
			if errors.Is(err, delivery.ErrWrongAddressPassword) {
				return s.reply(false, "nice try")
			}

			return err
		}

		if !l.lock(mailbox.ID) {
			return s.reply(false, "there is two of you?")
		}

		log.InfoContext(ctx).
			Int64("mailbox", mailbox.ID).
			Msg("locking mailbox")

		s.mailbox = mailbox
		s.inbox, err = inboxer.Inbox(ctx, mailbox)
		if err != nil {
			return err
		}

		s.state = sTransaction

		return s.reply(true, "I knew it was you!")
	}
}

// `QUIT` command as specified in RFC#1939
//
//     "QUIT" CRLF
func quit(inboxer *delivery.Inboxer) handler {
	return func(ctx context.Context, s *session, _ *command) error {
		if s.state.in(sTransaction) {
			if err := inboxer.Commit(ctx, s.mailbox, s.inbox); err != nil {
				return err
			}
		}

		log.DebugContext(ctx).Msg("closing session")
		return errCloseSession
	}
}

// `STAT` command as specified in RFC#1939
//
//     "STAT" CRLF
func stat() handler {
	return func(_ context.Context, s *session, _ *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		return s.reply(true, fmt.Sprintf("%d %d", s.inbox.Count(), s.inbox.Size()))
	}
}

// `LIST` command as specified in RFC#1939
//
//     "LIST" [ id ] CRLF
func list() handler {
	return func(_ context.Context, s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		switch len(c.args) {
		case 0:
			s.reply(true, fmt.Sprintf("%d messages (%d octets)", s.inbox.Count(), s.inbox.Size()))

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
			index, err := c.parseIndexArg(0)
			if err != nil {
				return errInvalidSyntax
			}

			mail, ok := s.inbox.Mail(index)
			if !ok {
				return s.reply(false, "no such message")
			}

			return s.reply(true, fmt.Sprintf("%d %d", index+1, mail.Size))

		default:
			return errInvalidSyntax
		}
	}
}

// `UIDL` command as specified in RFC#1939
//
//     "UIDL" [ id ] CRLF
func uidl() handler {
	return func(_ context.Context, s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		switch len(c.args) {
		case 0:
			s.reply(true, "list follows")

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
			index, err := c.parseIndexArg(0)
			if err != nil {
				return errInvalidSyntax
			}

			mail, ok := s.inbox.Mail(index)
			if !ok {
				return s.reply(false, "no such message")
			}

			return s.reply(true, fmt.Sprintf("%d %s", index+1, mail.ID))

		default:
			return errInvalidSyntax
		}
	}
}

// `RETR` command as specified in RFC#1939
//
//     "RETR" <id> CRLF
func retr(inboxer *delivery.Inboxer, blobs *storage.Blobs) handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		if len(c.args) != 1 {
			return errInvalidSyntax
		}

		index, err := c.parseIndexArg(0)
		if err != nil {
			return errInvalidSyntax
		}

		mail, ok := s.inbox.Mail(index)
		if !ok {
			return s.reply(false, "no such message")
		}

		if err := s.reply(true, "message coming"); err != nil {
			return err
		}

		log.InfoContext(ctx).
			Str("mail", mail.ID).
			Msg("retrieving mail")

		r, err := blobs.Reader(mail.ID)
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
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		if len(c.args) != 1 {
			return errInvalidSyntax
		}

		index, err := c.parseIndexArg(0)
		if err != nil {
			return errInvalidSyntax
		}

		if _, ok := s.inbox.Mail(index); !ok {
			return s.reply(false, "no such message")
		}

		s.inbox.Mark(index)

		log.InfoContext(ctx).
			Int("index", index).
			Msg("marking mail for deletion")

		return s.reply(true, "woosh")
	}
}

// `NOOP` command as specified in RFC#1939
//
//     "NOOP" CRLF
func noop() handler {
	return func(_ context.Context, s *session, _ *command) error {
		return s.reply(true, "what did you expect?")
	}
}

// `RSET` command as specified in RFC#1939
//
//     "RSET" CRLF
func rset() handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sTransaction) {
			return errBadSequence
		}

		s.inbox.Reset()

		log.InfoContext(ctx).Msg("resetting transaction")
		return s.reply(true, "lost some intel during time travel")
	}
}

// `STLS` command as specified in RFC#2595
//
//     "STLS" CRLF
func stls(config *tls.Config) handler {
	return func(ctx context.Context, s *session, _ *command) error {
		if config == nil {
			return s.reply(false, "I am afraid, I lost my disguise!")
		}

		if s.IsTLS() {
			return s.reply(false, "what are you afraid of?")
		}

		if err := s.reply(true, "ready to go undercover."); err != nil {
			return err
		}

		log.InfoContext(ctx).Msg("upgrading to tls")
		return s.UpgradeTLS(config)
	}
}

// `CAPA` command as specified in RFC#2449
//
//     "CAPA" CRLF
func capa(capabilities ...string) handler {
	return func(_ context.Context, s *session, _ *command) error {
		if err := s.reply(true, "I can do some things"); err != nil {
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
