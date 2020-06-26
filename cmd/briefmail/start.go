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

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/pop3"
	"github.com/lukasdietrich/briefmail/smtp"
	"github.com/lukasdietrich/briefmail/textproto"
)

type serverConfig struct {
	Address string
	TLS     bool
}

func init() {
	viper.SetDefault("smtp", []serverConfig{
		{Address: ":25"},
		{Address: ":587"},
	})

	viper.SetDefault("pop3", []serverConfig{
		{Address: ":110"},
		{Address: ":995", TLS: true},
	})
}

type startCommand struct {
	Smtp      *smtp.Proto
	Pop3      *pop3.Proto
	TlsConfig *tls.Config
}

func (s *startCommand) run() error {
	viper.Debug()

	startServers("smtp", s.Smtp, s.TlsConfig)
	startServers("pop3", s.Pop3, s.TlsConfig)

	return nil
}

func startServers(key string, proto textproto.Protocol, tlsConfig *tls.Config) {
	var configs []serverConfig
	viper.UnmarshalKey(key, &configs)

	for _, config := range configs {
		go startInstance(proto, config, tlsConfig)
	}
}

func startInstance(proto textproto.Protocol, serverConfig serverConfig, tlsConfig *tls.Config) {
	logrus.Infof("starting server on %s (force tls=%v)",
		serverConfig.Address, serverConfig.TLS)

	if !serverConfig.TLS {
		tlsConfig = nil
	}

	err := textproto.NewServer(proto, tlsConfig).Listen(serverConfig.Address)
	if err != nil {
		logrus.Fatal(err)
	}
}
