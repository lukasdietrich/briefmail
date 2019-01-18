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
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/lukasdietrich/briefmail/delivery"
	"github.com/lukasdietrich/briefmail/model"
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
	rOk := reply{250, "nothing happend. as expected"}

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

		s.envelope.From = nil
		s.envelope.To = nil

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
func mail() handler {
	rOk := reply{250, "noted."}

	return func(s *session, c *command) error {
		if !s.state.in(sHelo, sMail) {
			return errBadSequence
		}

		arg, _, err := c.args("FROM")
		if err != nil {
			return err
		}

		from, err := model.ParseAddress(arg)
		if err != nil {
			return err
		}

		s.envelope.From = from
		s.state = sMail

		return s.send(&rOk)
	}
}

// `RCPT` command as specified in RFC#5321 4.1.1.3
//
//     "RCPT TO:<" <Forward-path> ">" [ SP Paramters ] CRLF
func rcpt(mailman *delivery.Mailman) handler {
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

		to, err := model.ParseAddress(arg)
		if err != nil {
			return err
		}

		if !mailman.IsDeliverable(to, true) {
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
func data(mailman *delivery.Mailman) handler {
	var (
		rData = reply{354, "go ahead. period."}
		rOk   = reply{250, "confirmed transfer."}
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

		body := model.Body{Reader: s.DotReader()}
		body.Prepend("Received", fmt.Sprintf("from %s by (briefmail); %s",
			s.envelope.From,
			time.Now().Format(time.RFC1123Z)))

		if err := mailman.Deliver(&s.envelope, body); err != nil {
			return err
		}

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
