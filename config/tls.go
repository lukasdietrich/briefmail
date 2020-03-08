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

package config

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type certSource interface {
	time() (time.Time, error)
	load() (*tls.Certificate, error)
}

type TLS struct {
	Source string

	// source = "files"
	Crt string // *.crt pem encoded file
	Key string // *.key pem encoded file

	// source = "traefik"
	Acme   string // acme.json file from traefik
	Domain string // domain to search for in acme file
}

func (t *TLS) MakeTLSConfig() (*tls.Config, error) {
	var (
		log = logrus.WithField("prefix", "config-tls")

		source   certSource
		lastCert *tls.Certificate
		lastTime time.Time
		lock     sync.Mutex
	)

	switch strings.ToLower(t.Source) {
	case "files":
		log.Debugf("using certifcates from files: %s and %s", t.Crt, t.Key)
		source = &filesCertSource{
			crtFile: t.Crt,
			keyFile: t.Key,
		}

	case "traefik":
		log.Debugf("using certificates from traefik: %s for domain %s",
			t.Acme, t.Domain)

		source = &traefikCertSource{
			acmeFile: t.Acme,
			domain:   t.Domain,
		}

	case "":
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown certificate source: %s", t.Source)
	}

	return &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			lock.Lock()
			defer lock.Unlock()

			log.Debug("checking for certificate updates")

			newTime, err := source.time()
			if err != nil {
				return nil, err
			}

			if newTime.After(lastTime) {
				log.Debug("certificate source is updated")

				newCert, err := source.load()
				if err != nil {
					return nil, err
				}

				lastTime = newTime
				lastCert = newCert

				log.Debugf("reloaded certificate %s", lastTime)
			}

			return lastCert, nil
		},
	}, nil
}

type filesCertSource struct {
	crtFile string
	keyFile string
}

func (s *filesCertSource) time() (time.Time, error) {
	var t time.Time

	for _, file := range [...]string{s.crtFile, s.keyFile} {
		info, err := os.Stat(file)
		if err != nil {
			return t, err
		}

		if info.ModTime().After(t) {
			t = info.ModTime()
		}
	}

	return t, nil
}

func (s *filesCertSource) load() (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(s.crtFile, s.keyFile)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

type traefikCertSource struct {
	acmeFile string
	domain   string
}

func (s *traefikCertSource) time() (time.Time, error) {
	info, err := os.Stat(s.acmeFile)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

func (s *traefikCertSource) load() (*tls.Certificate, error) {
	type Entries []struct {
		Domain struct {
			Main string `json:"main"`
		} `json:"domain"`

		Crt string `json:"certificate"`
		Key string `json:"key"`
	}

	var data struct {
		Entries     Entries `json:"certificates"` // Traefik v1
		Letsencrypt struct {
			Entries Entries `json:"certificates"` // Traefik v2
		} `json:"letsencrypt"`
	}

	f, err := os.Open(s.acmeFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}

	for _, entry := range append(data.Entries, data.Letsencrypt.Entries...) {
		if entry.Domain.Main == s.domain {
			crtPem, err := base64.StdEncoding.DecodeString(entry.Crt)
			if err != nil {
				return nil, err
			}

			keyPem, err := base64.StdEncoding.DecodeString(entry.Key)
			if err != nil {
				return nil, err
			}

			cert, err := tls.X509KeyPair(crtPem, keyPem)
			if err != nil {
				return nil, err
			}

			return &cert, nil
		}
	}

	return nil, fmt.Errorf("no certificate for domain=%s found", s.domain)
}
