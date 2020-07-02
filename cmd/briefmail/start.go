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

package main

import (
	"crypto/tls"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/pop3"
	"github.com/lukasdietrich/briefmail/internal/smtp"
	"github.com/lukasdietrich/briefmail/internal/textproto"
)

type serverConfig struct {
	// Address is the port and optional host to bind a server to.
	Address string
	// TLS is a flag to indicate if the server should require a tls handshake
	// on inbound connections. If set to false, the client can still initiate
	// an upgrade during communication.
	TLS bool
}

// startCommand is a dependency container for the `briefmail start` command.
// It is used to wire the protocol implementations and tls configuration.
type startCommand struct {
	// SMTPProto is the protocol implementation for an smtp server.
	SMTPProto *smtp.Proto
	// POP3Proto is the protocol implementation for a pop3 server.
	POP3Proto *pop3.Proto
	// TLSConfig is either nil or wraps the configured tls certificate source.
	TLSConfig *tls.Config
}

// run starts smtp and pop3 servers on all configured ports.
func (s *startCommand) run() error {
	startServers("smtp", s.SMTPProto, s.TLSConfig)
	startServers("pop3", s.POP3Proto, s.TLSConfig)

	<-make(chan struct{})
	return nil
}

// startServers first determines all instance configs for a protocol and then
// starts a server for each entry.
func startServers(protoName string, proto textproto.Protocol, tlsConfig *tls.Config) error {
	configs, err := unmarshalServerConfigs(protoName)
	if err != nil {
		return fmt.Errorf("could not unmarshal %s server configuration: %w",
			protoName, err)
	}

	if len(configs) == 0 {
		logrus.Infof("no %s server configured", protoName)
	}

	for _, config := range configs {
		logrus.Infof("starting %s server on %q (tls=%t)",
			protoName, config.Address, config.TLS)

		go startInstance(proto, config, tlsConfig)
	}

	return nil
}

// unmarshalServerConfigs reads the config for either "pop3" or "smtp" and
// unmarshals it into a slice of serverConfig.
func unmarshalServerConfigs(protoName string) ([]serverConfig, error) {
	var configs []serverConfig
	return configs, viper.UnmarshalKey(protoName, &configs)
}

// startInstance creates a new server instance and listens on the configured port.
func startInstance(proto textproto.Protocol, config serverConfig, tlsConfig *tls.Config) {
	if !config.TLS {
		tlsConfig = nil
	}

	err := textproto.NewServer(proto, tlsConfig).Listen(config.Address)
	if err != nil {
		logrus.Fatal(err)
	}
}
