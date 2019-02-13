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
	"github.com/urfave/cli"

	"github.com/lukasdietrich/briefmail/addressbook"
	"github.com/lukasdietrich/briefmail/config"
	"github.com/lukasdietrich/briefmail/delivery"
	"github.com/lukasdietrich/briefmail/pop3"
	"github.com/lukasdietrich/briefmail/smtp"
	"github.com/lukasdietrich/briefmail/textproto"
)

func start() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Start smtp and pop3 servers",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config",
				Value: "config.toml",
			},
			cli.StringFlag{
				Name:  "addressbook",
				Value: "addressbook.toml",
			},
		},
		Action: func(ctx *cli.Context) error {
			config, err := config.Parse(ctx.String("config"))
			if err != nil {
				return err
			}

			domains, err := config.General.NormalizedDomains()
			if err != nil {
				return err
			}

			book, err := addressbook.Parse(ctx.String("addressbook"), DB)
			if err != nil {
				return err
			}

			book.SetLocalDomains(domains)

			queue := delivery.QueueWorker{
				DB:    DB,
				Blobs: Blobs,
			}

			mailman := delivery.Mailman{
				DB:    DB,
				Blobs: Blobs,
				Book:  book,
				Queue: &queue,
			}

			tlsConfig, err := config.TLS.MakeTLSConfig()
			if err != nil {
				return err
			}

			fromHooks, dataHooks, err := config.Hook.MakeInstances()
			if err != nil {
				return err
			}

			for _, instance := range config.Smtp {
				go startServer(smtp.New(&smtp.Config{
					Hostname:  config.General.Hostname,
					MaxSize:   config.Mail.Size,
					Mailman:   &mailman,
					Book:      book,
					Cache:     Cache,
					DB:        DB,
					TLS:       tlsConfig,
					FromHooks: fromHooks,
					DataHooks: dataHooks,
				}), instance.Address, tlsConfig, instance.TLS)

				logrus.Infof("start smtp @ %s (tls: %v)",
					instance.Address, instance.TLS)
			}

			for _, instance := range config.Pop3 {
				go startServer(pop3.New(&pop3.Config{
					Hostname: config.General.Hostname,
					DB:       DB,
					Blobs:    Blobs,
					TLS:      tlsConfig,
				}), instance.Address, tlsConfig, instance.TLS)

				logrus.Infof("start pop3 @ %s (tls: %v)",
					instance.Address, instance.TLS)
			}

			queue.WakeUp()

			<-make(chan bool)
			return nil
		},
	}
}

func startServer(
	proto textproto.Protocol,
	addr string,
	tlsConfig *tls.Config,
	tlsOnly bool,
) {
	if !tlsOnly {
		tlsConfig = nil
	}

	if err := textproto.NewServer(proto, tlsConfig).Listen(addr); err != nil {
		logrus.Fatal(err)
	}
}
