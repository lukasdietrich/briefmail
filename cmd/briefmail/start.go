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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	servers := instanceManager{
		smtpProto: s.SMTPProto,
		pop3Proto: s.POP3Proto,
		tlsConfig: s.TLSConfig,
	}

	if err := servers.start(); err != nil {
		return err
	}

	s.handleSignals(&servers)
	return nil
}

// handleSignals waits for SIGINT or SIGTERM and then tries to gracefully
// shutdown all servers. If another signal is captured, the shutdown will be
// forced immediately.
func (s *startCommand) handleSignals(servers *instanceManager) {
	const timeout = time.Second * 30

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	logrus.Info("waiting for shutdown signals..")
	<-signals

	logrus.Info("trying to shutdown gracefully")

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	go servers.shutdown(ctx, cancelFunc)

	select {
	case <-signals:
		logrus.Info("shutting down forcefully now")
		cancelFunc()

	case <-ctx.Done():
	}
}

// instanceManager is a container for all configured server instances.
// It also keeps track of how many servers are still running.
type instanceManager struct {
	smtpProto textproto.Protocol
	pop3Proto textproto.Protocol
	tlsConfig *tls.Config
	servers   []textproto.Server
	wg        sync.WaitGroup
}

// shutdown tries to gracefully shutdown all started server instances.
func (i *instanceManager) shutdown(ctx context.Context, cancelFunc context.CancelFunc) {
	for _, server := range i.servers {
		go i.shutdownInstance(ctx, server)
	}

	i.wg.Wait()
	logrus.Info("all servers stopped gracefully")
	cancelFunc()
}

// shutdownInstance tries to gracefully shutdown a single server instance.
func (i *instanceManager) shutdownInstance(ctx context.Context, server textproto.Server) {
	server.Shutdown(ctx)
	i.wg.Done()
}

// start reads all configured smtp and pop3 servers and then starts all of them.
func (i *instanceManager) start() error {
	for protoName, proto := range map[string]textproto.Protocol{
		"smtp": i.smtpProto,
		"pop3": i.pop3Proto,
	} {
		configSlice, err := unmarshalServerConfigs(protoName)
		if err != nil {
			return fmt.Errorf("could not unmarshal %s server configuration: %w",
				protoName, err)
		}

		if len(configSlice) == 0 {
			logrus.Warnf("%s is not configured", protoName)
			continue
		}

		for _, config := range configSlice {
			logrus.Infof("starting %s server on %q (tls=%t)",
				protoName, config.Address, config.TLS)

			var tlsConfig *tls.Config
			if config.TLS {
				tlsConfig = i.tlsConfig
			}

			server := textproto.NewServer(proto, tlsConfig)
			i.servers = append(i.servers, server)
			go i.startInstance(server, config.Address)
		}
	}

	i.wg.Add(len(i.servers))
	return nil
}

// startInstance starts a single server instance. All errors except
// textproto.ErrServerClosed are logged and cause a panic.
func (i *instanceManager) startInstance(server textproto.Server, addr string) {
	if err := server.Listen(addr); err != nil {
		if !errors.Is(err, textproto.ErrServerClosed) {
			logrus.Fatal(err)
		}
	}

	logrus.Infof("server %s stopped", addr)
}

// unmarshalServerConfigs reads the config for either "pop3" or "smtp" and
// unmarshals it into a slice of serverConfig.
func unmarshalServerConfigs(protoName string) ([]serverConfig, error) {
	logrus.Debugf("reading %s configuration", protoName)

	var configs []serverConfig
	return configs, viper.UnmarshalKey(protoName, &configs)
}
