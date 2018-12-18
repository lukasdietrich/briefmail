package smtp

import (
	"time"

	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/textproto"
)

type sessionState uint

const (
	sInit sessionState = iota
	sHelo
	sMail
	sRcpt
	sData
)

func (s sessionState) String() string {
	return [...]string{
		"init",
		"helo",
		"mail",
		"rcpt",
		"data",
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
	envelope model.Envelope
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
