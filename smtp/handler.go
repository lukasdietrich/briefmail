package smtp

import (
	"errors"
	"time"

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
	return func(s *session, _ *command) error {
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
func rcpt() handler {
	rOk := reply{250, "yup, another?"}

	return func(s *session, c *command) error {
		if !s.state.in(sMail, sRcpt) {
			return errBadSequence
		}

		arg, _, err := c.args("TO")
		if err != nil {
			return err
		}

		to, err := model.ParseAddress(arg)
		if err != nil {
			return err
		}

		s.envelope.To = append(s.envelope.To, to)
		s.state = sRcpt

		return s.send(&rOk)
	}
}

// `DATA` command as specified in RFC#5321 4.1.1.4
//
//     "DATA" CRLF
func data(deliver DeliverFunc) handler {
	var (
		rData = reply{354, "go ahead. period."}
		rOk   = reply{250, "confirmed transfer."}
	)

	return func(s *session, c *command) error {
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

		if err := deliver(&s.envelope, s.DotReader()); err != nil {
			return err
		}

		s.state = sHelo
		return s.send(&rOk)
	}
}
