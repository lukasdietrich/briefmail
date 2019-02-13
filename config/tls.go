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
)

type certFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

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
	var f certFunc

	switch strings.ToLower(t.Source) {
	case "files":
		f = certByFile(t.Crt, t.Key)

	case "traefik":
		f = certByTraefikJson(t.Acme, t.Domain)

	case "":
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown certificate source: %s", t.Source)
	}

	return &tls.Config{GetCertificate: f}, nil
}

func certByFile(crtFile, keyFile string) certFunc {
	var (
		files    = []string{crtFile, keyFile}
		mutex    sync.Mutex
		lastCert *tls.Certificate
		lastTime time.Time
	)

	return func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		mutex.Lock()
		defer mutex.Unlock()

		if lastCert != nil {
			for _, file := range files {
				info, err := os.Stat(file)
				if err != nil {
					return nil, err
				}

				if info.ModTime().After(lastTime) {
					goto load
				}
			}

			return lastCert, nil
		}

	load:
		cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			return nil, err
		}

		lastCert = &cert
		lastTime = time.Now()

		return lastCert, nil
	}
}

func certByTraefikJson(acmeFile string, domain string) certFunc {
	var (
		base64   = base64.StdEncoding
		mutex    sync.Mutex
		lastCert *tls.Certificate
		lastTime time.Time
	)

	return func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		mutex.Lock()
		defer mutex.Unlock()

		if lastCert != nil {
			info, err := os.Stat(acmeFile)
			if err != nil {
				return nil, err
			}

			if info.ModTime().After(lastTime) {
				goto load
			}

			return lastCert, nil
		}

	load:
		var data struct {
			Entries []struct {
				Domain struct {
					Main string `json:"main"`
				} `json:"domain"`

				Pem string `json:"certificate"`
				Key string `json:"key"`
			} `json:"certificates"`
		}

		f, err := os.Open(acmeFile)
		if err != nil {
			return nil, err
		}

		defer f.Close()

		if err := json.NewDecoder(f).Decode(&data); err != nil {
			return nil, err
		}

		for _, entry := range data.Entries {
			if entry.Domain.Main == domain {
				crtPem, err := base64.DecodeString(entry.Pem)
				if err != nil {
					return nil, err
				}

				keyPem, err := base64.DecodeString(entry.Key)
				if err != nil {
					return nil, err
				}

				cert, err := tls.X509KeyPair(crtPem, keyPem)
				if err != nil {
					return nil, err
				}

				lastCert = &cert
				lastTime = time.Now()

				return lastCert, nil
			}
		}

		return nil, fmt.Errorf("no certificate for domain=%s found", domain)
	}
}
