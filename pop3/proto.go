package pop3

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/lukasdietrich/briefmail/storage"
	"github.com/lukasdietrich/briefmail/textproto"
)

type Config struct {
	Hostname string
	DB       *storage.DB
	Blobs    *storage.Blobs
	TLS      *tls.Config
}

type proto struct {
	config     *Config
	locks      *locks
	handlerMap map[string]handler
}

// New creates a new Protocol instance to be used with a textproto Server
func New(config *Config) textproto.Protocol {
	locks := newLocks()

	return &proto{
		config: config,
		locks:  locks,
		handlerMap: map[string]handler{
			"USER": user(),
			"PASS": pass(locks, config.DB),

			"STAT": stat(),
			"LIST": list(),
			"RETR": retr(config.Blobs),
			"DELE": dele(),

			"NOOP": noop(),
			"RSET": rset(),
			"QUIT": quit(config.DB),

			"STLS": stls(config.TLS),
		},
	}
}

var (
	rReady          = reply{true, "ready"}
	rBye            = reply{true, "closing transmission channel"}
	rTimeout        = reply{false, "timed out"}
	rError          = reply{false, "action aborted: local error in processing"}
	rNotImplemented = reply{false, "command not implemented"}
	rBadSequence    = reply{false, "bad sequence of commands"}
	rInvalidSyntax  = reply{false, "invalid syntax"}
)

func (p *proto) Handle(c textproto.Conn) {
	s := &session{
		Conn:  c,
		state: sInit,
	}

	if err := s.send(&rReady); err != nil {
		return
	}

	switch err := p.loop(s); err {
	case io.EOF, errCloseSession, nil:
		s.send(&rBye)
	default:
		logrus.Warn(err)

		if errt, ok := err.(*net.OpError); ok && errt.Timeout() {
			s.send(&rTimeout)
		} else {
			s.send(&rError)
		}
	}

	if s.state == sTransaction {
		p.locks.unlock(s.mailbox.id)
	}
}

func (p *proto) loop(s *session) error {
	var cmd command

	for {
		if err := s.read(&cmd); err != nil {
			return err
		}

		h, ok := p.handlerMap[string(bytes.ToUpper(cmd.head))]

		if !ok {
			if err := s.send(&rNotImplemented); err != nil {
				return err
			}

			continue
		}

		if err := h(s, &cmd); err != nil {
			switch err {
			case errBadSequence:
				if err := s.send(&rBadSequence); err != nil {
					return err
				}

			case errInvalidSyntax:
				if err := s.send(&rInvalidSyntax); err != nil {
					return err
				}

			default:
				return err
			}
		}
	}
}
