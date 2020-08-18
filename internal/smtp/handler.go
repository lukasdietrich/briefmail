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
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var (
	errCloseSession = errors.New("smtp: session closed")
	errBadSequence  = errors.New("smtp: bad sequence of commands")
)

type handler func(*session, *command) error

// `HELO` command as specified in RFC#5321 4.1.1.1
//
//     "HELO" SP <Domain> CRLF
func helo(hostname string) handler {
	rReady := reply{250, hostname}

	return func(s *session, c *command) error {
		s.state = sHelo
		s.envelope.Helo = string(c.tail)

		return s.send(&rReady)
	}
}

// `EHLO` command as specified in RFC#5321 4.1.1.1
//
//     "EHLO" SP <Domain OR address-literal> CRLF
func ehlo(hostname string, extensions ...string) handler {
	extensions = append(extensions, "8BITMIME")

	// nolint:errcheck
	return func(s *session, c *command) error {
		s.state = sHelo
		s.envelope.Helo = string(c.tail)

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
	rOk := reply{250, "nothing happened. as expected"}

	return func(s *session, _ *command) error {
		return s.send(&rOk)
	}
}

// `RSET` command as specified in RFC#5321 4.1.1.5
//
//     "RSET" CRLF
func rset() handler {
	rOk := reply{250, "everything gone. pinky promise"}

	return func(s *session, _ *command) error {
		if !s.state.in(sInit, sHelo) {
			s.state = sHelo
		}

		s.envelope.From = mails.ZeroAddress
		s.envelope.To = nil
		s.headers = nil

		return s.send(&rOk)
	}
}

// `VRFY` command as specified in RFC#5321 4.1.1.6
//
//     "VRFY" SP <user OR mailbox> CRLF
func vrfy() handler {
	rMaybe := reply{252, "maybe, maybe not? who knows for sure"}

	return func(s *session, _ *command) error {
		return s.send(&rMaybe)
	}
}

// `QUIT` command as specified in RFC#5321 4.1.1.10
//
//     "QUIT" CRLF
func quit() handler {
	return func(*session, *command) error {
		return errCloseSession
	}
}

// `MAIL` command as specified in RFC#5321 4.1.1.2
//
//     "MAIL FROM:<" <Reverse-path> ">" [ SP Parameters ] CRLF
func mail(addressbook *delivery.Addressbook, maxSize int64, hooks []hook.FromHook) handler {
	var (
		rOk   = reply{250, "noted."}
		rSize = reply{552, "bit too much"}
		rAuth = reply{530, "that does not sound like you"}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sHelo, sMail) {
			return errBadSequence
		}

		arg, params, err := c.args("FROM")
		if err != nil {
			return err
		}

		from, err := mails.ParseUnicode(arg)
		if err != nil {
			return err
		}

		origin, err := addressbook.Lookup(s.Context(), from)
		if err != nil {
			return err
		}

		if s.isSubmission() {
			// authenticated connections must send mails from a local address,
			// which the current user owns

			if !origin.IsLocal || origin.Mailbox.ID != s.mailbox.ID {
				return s.send(&rAuth)
			}
		} else {
			// unauthenticated connections must send mails from a remote address

			if origin.IsLocal {
				return s.send(&rAuth)
			}
		}

		// see RFC#1870 "6. The extended MAIL command"
		if maxSize > 0 {
			if size, ok := params["SIZE"]; ok {
				isize, err := strconv.ParseInt(size, 10, 64)
				if err != nil {
					return errCommandSyntax
				}

				if isize > maxSize {
					return s.send(&rSize)
				}
			}
		}

		var headers []hook.HeaderField

		for _, hook := range hooks {
			result, err := hook(s.isSubmission(), s.RemoteAddr(), from)
			if err != nil {
				return err
			}

			if result.Reject {
				return s.send(&reply{result.Code, result.Text})
			}

			headers = append(headers, result.Headers...)
		}

		s.headers = headers
		s.envelope.From = from
		s.state = sMail

		return s.send(&rOk)
	}
}

// `RCPT` command as specified in RFC#5321 4.1.1.3
//
//     "RCPT TO:<" <Forward-path> ">" [ SP Parameters ] CRLF
func rcpt(addressbook *delivery.Addressbook) handler {
	var (
		rOk                = reply{250, "yup, another?"}
		rTooManyRecipients = reply{452, "that is quite a crowd already!"}
		rInvalidRecipient  = reply{550, "never heard of that person."}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sMail, sRcpt) {
			return errBadSequence
		}

		if len(s.envelope.To) > 100 {
			return s.send(&rTooManyRecipients)
		}

		arg, _, err := c.args("TO")
		if err != nil {
			return err
		}

		to, err := mails.ParseUnicode(arg)
		if err != nil {
			return err
		}

		destination, err := addressbook.Lookup(s.Context(), to)
		if err != nil {
			return err
		}

		if (s.isSubmission() && !destination.IsLocal) || destination.Mailbox == nil {
			return s.send(&rInvalidRecipient)
		}

		s.envelope.To = append(s.envelope.To, to)
		s.state = sRcpt

		return s.send(&rOk)
	}
}

// `DATA` command as specified in RFC#5321 4.1.1.4
//
//     "DATA" CRLF
func data(mailman *delivery.Mailman, cache *storage.Cache, maxSize int64, hooks []hook.DataHook) handler {
	var (
		rData = reply{354, "go ahead. period."}
		rOk   = reply{250, "confirmed transfer."}
		rSize = reply{552, "I am already full, thanks"}
	)

	return func(s *session, _ *command) error {
		if !s.state.in(sRcpt) {
			return errBadSequence
		}

		if err := s.send(&rData); err != nil {
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

		entry, err := cache.Write(prepender.reader(lr))
		if err != nil {
			if err == errReaderLimitReached {
				// discard remaining bytes (but not forever) to flush
				// the input stream
				_, err := io.Copy(ioutil.Discard, &limitedReader{r, maxSize})
				if err != nil {
					return err
				}

				return s.send(&rSize)
			}

			return err
		}

		defer entry.Release()

		var headers []hook.HeaderField

		for _, hook := range hooks {
			r, err := entry.Reader()
			if err != nil {
				return err
			}

			result, err := hook(s.isSubmission(), r)
			if err != nil {
				return err
			}

			if result.Reject {
				return s.send(&reply{result.Code, result.Text})
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

		if err := mailman.Deliver(s.Context(), s.envelope, content); err != nil {
			return err
		}

		log.WithField("from", s.envelope.From).
			Debug("mail successfully received")

		s.state = sHelo
		return s.send(&rOk)
	}
}

// `STARTTLS` command as specified in RFC#3207
//
//     "STARTTLS" CRLF
func starttls(config *tls.Config) handler {
	var (
		rReady          = reply{220, "ready to go undercover."}
		rTLSUnavailable = reply{454, "I am afraid, I lost my disguise!"}
		rAlreadyTLS     = reply{454, "what are you afraid of?"}
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

// `AUTH` command as specified in RFC#4954
//
//     "AUTH" <Mechanism> [ Payload ] CRLF
func auth(authenticator *delivery.Authenticator) handler {
	var (
		rUsername = reply{334, "VXNlcm5hbWU6"}
		rPassword = reply{334, "UGFzc3dvcmQ6"}
		rOk       = reply{235, "I was sure I saw you before."}
		rFail     = reply{535, "Solid attempt."}
	)

	return func(s *session, c *command) error {
		if !s.state.in(sHelo) {
			return errBadSequence
		}

		var (
			fields     = bytes.Fields(c.tail)
			name, pass []byte
		)

		if len(fields) < 1 {
			return errCommandSyntax
		}

		switch strings.ToUpper(string(fields[0])) {
		case "PLAIN":
			if len(fields) != 2 {
				return errCommandSyntax
			}

			b, err := base64.StdEncoding.DecodeString(string(fields[1]))
			if err != nil {
				return errCommandSyntax
			}

			fields = bytes.Split(b, []byte{0})
			if len(fields) != 3 {
				return errCommandSyntax
			}

			if len(fields[0]) > 0 {
				if !bytes.Equal(fields[0], fields[1]) {
					return s.send(&rFail)
				}
			}

			name = fields[1]
			pass = fields[2]

		case "LOGIN":
			if err := s.send(&rUsername); err != nil {
				return err
			}

			b, err := s.ReadLine()
			if err != nil {
				return err
			}

			b, err = base64.StdEncoding.DecodeString(string(b))
			if err != nil {
				return errCommandSyntax
			}

			name = b

			if err := s.send(&rPassword); err != nil {
				return err
			}

			b, err = s.ReadLine()
			if err != nil {
				return err
			}

			pass, err = base64.StdEncoding.DecodeString(string(b))
			if err != nil {
				return errCommandSyntax
			}

		default:
			return errCommandSyntax
		}

		mailbox, err := authenticator.Auth(s.Context(), name, pass)
		if err != nil {
			if errors.Is(err, delivery.ErrWrongAddressPassword) {
				return s.send(&rFail)
			}

			return err
		}

		s.mailbox = mailbox
		return s.send(&rOk)
	}
}
