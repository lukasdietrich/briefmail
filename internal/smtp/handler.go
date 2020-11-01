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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var (
	errCloseSession = errors.New("smtp: session closed")
	errBadSequence  = errors.New("smtp: bad sequence of commands")
)

type handler func(context.Context, *session, *command) error

type smtpError struct {
	code  int
	text  string
	cause error
}

func (e smtpError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%v: %d %s", e.cause, e.code, e.text)
	}

	return fmt.Sprintf("%d %s", e.code, e.text)
}

// `HELO` command as specified in RFC#5321 4.1.1.1
//
//     "HELO" SP <Domain> CRLF
func helo(hostname string) handler {
	return func(ctx context.Context, s *session, c *command) error {
		s.state = sHelo
		s.envelope.Helo = string(c.tail)

		log.DebugContext(ctx).
			Str("hostname", s.envelope.Helo).
			Msg("resetting transaction state")

		return s.reply(250, hostname)
	}
}

// `EHLO` command as specified in RFC#5321 4.1.1.1
//
//     "EHLO" SP <Domain OR address-literal> CRLF
func ehlo(hostname string, extensions ...string) handler {
	extensions = append(extensions, "8BITMIME")

	return func(ctx context.Context, s *session, c *command) error {
		s.state = sHelo
		s.envelope.Helo = string(c.tail)

		log.DebugContext(ctx).
			Str("hostname", s.envelope.Helo).
			Msg("resetting transaction state")

		s.SetWriteTimeout(time.Minute * 5)

		s.WriteString("250-")
		s.WriteString(hostname)
		s.Endline()

		for _, ext := range extensions[1:] {
			s.WriteString("250-")
			s.WriteString(ext)
			s.Endline()
		}

		s.WriteString("250 ")
		s.WriteString(extensions[0])
		s.Endline()

		return s.Flush()
	}
}

// `NOOP` command as specified in RFC#5321 4.1.1.9
//
//     "NOOP" CRLF
func noop() handler {
	return func(_ context.Context, s *session, _ *command) error {
		return s.reply(250, "nothing happened. as expected")
	}
}

// `RSET` command as specified in RFC#5321 4.1.1.5
//
//     "RSET" CRLF
func rset() handler {
	return func(ctx context.Context, s *session, _ *command) error {
		if !s.state.in(sInit, sHelo) {
			s.state = sHelo
		}

		s.envelope.From = models.ZeroAddress
		s.envelope.To = nil
		s.headers = nil

		log.DebugContext(ctx).Msg("resetting transaction state")

		return s.reply(250, "everything gone. pinky promise")
	}
}

// `VRFY` command as specified in RFC#5321 4.1.1.6
//
//     "VRFY" SP <user OR mailbox> CRLF
func vrfy() handler {
	return func(_ context.Context, s *session, _ *command) error {
		return s.reply(252, "maybe, maybe not? who knows for sure")
	}
}

// `QUIT` command as specified in RFC#5321 4.1.1.10
//
//     "QUIT" CRLF
func quit() handler {
	return func(ctx context.Context, s *session, _ *command) error {
		log.DebugContext(ctx).Msg("closing session")
		return errCloseSession
	}
}

// `MAIL` command as specified in RFC#5321 4.1.1.2
//
//     "MAIL FROM:<" <Reverse-path> ">" [ SP Parameters ] CRLF
func mail(addressbook *delivery.Addressbook, maxSize int64, hooks []hook.FromHook) handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sHelo, sMail) {
			return errBadSequence
		}

		arg, params, err := c.args("FROM")
		if err != nil {
			return err
		}

		from, err := models.ParseUnicode(arg)
		if err != nil {
			return err
		}

		origin, err := addressbook.Lookup(ctx, from)
		if err != nil {
			return err
		}

		if err := checkOrigin(ctx, s, origin); err != nil {
			return err
		}

		if err := checkMaxSize(ctx, s, params, maxSize); err != nil {
			return err
		}

		if err := execFromHooks(ctx, s, from, hooks); err != nil {
			return err
		}

		s.envelope.From = from
		s.state = sMail

		log.DebugContext(ctx).
			Str("from", arg).
			Msg("beginning mail transaction")

		return s.reply(250, "noted.")
	}
}

func checkOrigin(ctx context.Context, s *session, origin *delivery.LookupResult) error {
	if s.isSubmission() {
		// authenticated connections must send mails from a local address,
		// which the current user owns

		if !origin.IsLocal || origin.Mailbox.ID != s.mailbox.ID {
			log.WarnContext(ctx).
				Int64("mailbox", s.mailbox.ID).
				Stringer("from", origin.Address).
				Msg("authenticated connection trying to send as someone else")

			return smtpError{code: 550, text: "that does not sound like you"}
		}
	} else {
		// unauthenticated connections must send mails from a remote address

		if origin.IsLocal {
			log.WarnContext(ctx).
				Stringer("from", origin.Address).
				Msg("attempted submission without authentication")
			return smtpError{code: 550, text: "submissions must be authenticated"}
		}
	}

	return nil
}

func checkMaxSize(ctx context.Context, s *session, params map[string]string, maxSize int64) error {
	// see RFC#1870 "6. The extended MAIL command"

	if maxSize == 0 {
		return nil
	}

	if sizeParam, ok := params["SIZE"]; ok {
		size, err := strconv.ParseInt(sizeParam, 10, 64)
		if err != nil {
			log.DebugContext(ctx).
				Str("size", sizeParam).
				Msg("invalid SIZE parameter")
			return errCommandSyntax
		}

		if size > maxSize {
			log.InfoContext(ctx).
				Int64("size", size).
				Int64("maxSize", maxSize).
				Msg("requested SIZE parameter execeeding maximum configured size")
			return smtpError{code: 552, text: "that is a bit too much"}
		}
	}

	return nil
}

func execFromHooks(ctx context.Context, s *session, from models.Address, hooks []hook.FromHook) error {
	var headers []hook.HeaderField

	for _, hook := range hooks {
		result, err := hook(ctx, s.isSubmission(), s.RemoteAddr(), from)
		if err != nil {
			return err
		}

		if result.Reject {
			return smtpError{code: result.Code, text: result.Text}
		}

		headers = append(headers, result.Headers...)
	}

	s.headers = headers
	return nil
}

// `RCPT` command as specified in RFC#5321 4.1.1.3
//
//     "RCPT TO:<" <Forward-path> ">" [ SP Parameters ] CRLF
func rcpt(addressbook *delivery.Addressbook) handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sMail, sRcpt) {
			return errBadSequence
		}

		arg, _, err := c.args("TO")
		if err != nil {
			return err
		}

		if len(s.envelope.To) > 100 {
			log.DebugContext(ctx).
				Int("recipientCount", len(s.envelope.To)).
				Msg("too many recipients")
			return s.reply(452, "that is quite a crowd already!")
		}

		to, err := models.ParseUnicode(arg)
		if err != nil {
			return err
		}

		destination, err := addressbook.Lookup(ctx, to)
		if err != nil {
			return err
		}

		if !isValidDestination(s, destination) {
			log.DebugContext(ctx).
				Stringer("to", to).
				Msg("invalid recipient")

			return s.reply(550, "never heard of that person.")
		}

		s.envelope.To = append(s.envelope.To, to)
		s.state = sRcpt

		log.DebugContext(ctx).
			Str("to", arg).
			Msg("recipient added")

		return s.reply(250, "yup, another?")
	}
}

func isValidDestination(s *session, destination *delivery.LookupResult) bool {
	if destination.IsLocal {
		// when the destination is a local mailbox, the mailbox must exist.
		return destination.Mailbox != nil
	}

	// when the destination is outbound, it must be an authenticated connection (= submission)
	return s.isSubmission()
}

// `DATA` command as specified in RFC#5321 4.1.1.4
//
//     "DATA" CRLF
func data(mailman *delivery.Mailman, cache *storage.Cache, maxSize int64, hooks []hook.DataHook) handler {
	return func(ctx context.Context, s *session, _ *command) error {
		if !s.state.in(sRcpt) {
			return errBadSequence
		}

		log.DebugContext(ctx).Msg("receiving mail content")

		if err := s.reply(354, "go ahead. period."); err != nil {
			return err
		}

		if err := s.SetReadTimeout(time.Minute * 10); err != nil {
			return err
		}

		s.envelope.Date = time.Now()

		var (
			r  = s.DotReader()
			lr = r
		)

		if maxSize > 0 {
			// limit reader to the allowed size plus a little extra
			lr = &limitedReader{r, maxSize + 1024}
		}

		prepender := newPrepender(8)
		prepender.prepend("Received", fmt.Sprintf("from %s by (briefmail); %s",
			s.envelope.From,
			s.envelope.Date.Format(time.RFC1123Z)))

		for _, header := range s.headers {
			prepender.prepend(header.Key, header.Value)
		}

		entry, err := cache.Write(ctx, prepender.reader(lr))
		if err != nil {
			if err == errReaderLimitReached {
				// discard remaining bytes (but not forever) to flush
				// the input stream
				_, err := io.Copy(ioutil.Discard, &limitedReader{r, maxSize})
				if err != nil {
					return err
				}

				return s.reply(552, "I am already full, thanks")
			}

			return err
		}

		defer entry.Release(ctx)

		var headers []hook.HeaderField

		for _, hook := range hooks {
			r, err := entry.Reader()
			if err != nil {
				return err
			}

			result, err := hook(ctx, s.isSubmission(), r)
			if err != nil {
				return err
			}

			if result.Reject {
				return s.reply(result.Code, result.Text)
			}

			headers = append(headers, result.Headers...)
		}

		r, err = entry.Reader()
		if err != nil {
			return err
		}

		prepender.reset()

		for _, header := range headers {
			prepender.prepend(header.Key, header.Value)
		}

		content := prepender.reader(r)

		log.InfoContext(ctx).Msg("committing mail transaction")

		if err := mailman.Deliver(ctx, s.envelope, content); err != nil {
			return err
		}

		s.state = sHelo
		return s.reply(250, "confirmed transfer.")
	}
}

// `STARTTLS` command as specified in RFC#3207
//
//     "STARTTLS" CRLF
func starttls(config *tls.Config) handler {
	return func(ctx context.Context, s *session, _ *command) error {
		if config == nil {
			return s.reply(454, "I am afraid, I lost my disguise!")
		}

		if s.IsTLS() {
			return s.reply(454, "what are you afraid of?")
		}

		if err := s.reply(220, "ready to go undercover."); err != nil {
			return err
		}

		return s.UpgradeTLS(config)
	}
}

// `AUTH` command as specified in RFC#4954
//
//     "AUTH" <Mechanism> [ Payload ] CRLF
func auth(authenticator *delivery.Authenticator) handler {
	return func(ctx context.Context, s *session, c *command) error {
		if !s.state.in(sHelo) {
			return errBadSequence
		}

		name, pass, err := determineNamePass(s, c)
		if err != nil {
			return err
		}

		mailbox, err := authenticator.Auth(ctx, name, pass)
		if err != nil {
			if errors.Is(err, delivery.ErrWrongAddressPassword) {
				return s.reply(535, "Solid attempt.")
			}

			return err
		}

		s.mailbox = mailbox
		return s.reply(235, "I was sure I saw you before.")
	}
}

func determineNamePass(s *session, c *command) (name, pass []byte, err error) {
	if len(c.tail) == 0 {
		return nil, nil, errCommandSyntax
	}

	var mechanism string

	space := bytes.IndexByte(c.tail, ' ')
	if space < 0 {
		mechanism = string(c.tail)
	} else {
		mechanism = string(c.tail[:space])
	}

	switch strings.ToLower(mechanism) {
	case "plain":
		return parsePlainAuth(c.tail[space+1:])

	case "login":
		return processLoginAuth(s)

	default:
		return nil, nil, errCommandSyntax
	}
}

func parsePlainAuth(tail []byte) (name, pass []byte, err error) {
	if len(tail) != 1 {
		return nil, nil, errCommandSyntax
	}

	b, err := decodeBase64Bytes(tail)
	if err != nil {
		return nil, nil, errCommandSyntax
	}

	switch fields := bytes.Split(b, []byte{0}); len(fields) {
	case 2:
		// <authentication-identity> NULLBYTE <password>
		return fields[0], fields[1], nil

	case 3:
		// <authorization-identity> NULLBYTE <authentication-identity> NULLBYTE <password>
		// we only accept authorization == authentication

		if !bytes.Equal(fields[0], fields[1]) {
			return nil, nil, errCommandSyntax
		}

		return fields[0], fields[2], nil

	default:
		return nil, nil, errCommandSyntax
	}
}

func processLoginAuth(s *session) (name, pass []byte, err error) {
	if err := s.reply(334, "VXNlcm5hbWU6"); err != nil {
		return nil, nil, err
	}

	b, err := s.ReadLine()
	if err != nil {
		return nil, nil, err
	}

	name, err = decodeBase64Bytes(b)
	if err != nil {
		return nil, nil, errCommandSyntax
	}

	if err := s.reply(334, "UGFzc3dvcmQ6"); err != nil {
		return nil, nil, err
	}

	b, err = s.ReadLine()
	if err != nil {
		return nil, nil, err
	}

	pass, err = decodeBase64Bytes(b)
	if err != nil {
		return nil, nil, errCommandSyntax
	}

	return
}

func decodeBase64Bytes(encoded []byte) ([]byte, error) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		return nil, err
	}

	return decoded[:n], nil
}
