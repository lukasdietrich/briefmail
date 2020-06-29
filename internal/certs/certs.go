// Copyright (C) 2020  Lukas Dietrich <lukas@lukasdietrich.com>
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

package certs

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	sourceNone    = "none"
	sourceFiles   = "files"
	sourceTraefik = "traefik"
)

var log = logrus.WithField("prefix", "certs")

func init() {
	viper.SetDefault("tls.source", sourceNone)
}

type CertSource interface {
	LastUpdate() (time.Time, error)
	Load() (*tls.Certificate, error)
}

func NewCertSource() (CertSource, error) {
	switch source := viper.GetString("tls.source"); source {
	case sourceNone:
		return nil, nil
	case sourceFiles:
		return newFilesCertSource(), nil
	case sourceTraefik:
		return newTraefikCertSource(), nil
	default:
		return nil, fmt.Errorf("unknown certificate source: %s", source)
	}
}

func NewTlsConfig(source CertSource) *tls.Config {
	var (
		lastCert *tls.Certificate
		lastTime time.Time
		lock     sync.Mutex
	)

	return &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			lock.Lock()
			defer lock.Unlock()

			newTime, err := source.LastUpdate()
			if err != nil {
				log.Errorf("could not check for certificate updates: %v", err)
				return nil, err
			}

			if newTime.After(lastTime) {
				newCert, err := source.Load()
				if err != nil {
					log.Errorf("could not load certificate: %v", err)
					return nil, err
				}

				lastTime = newTime
				lastCert = newCert

				log.Debugf("loaded certificate from %s", lastTime)
			}

			return lastCert, nil
		},
	}
}
