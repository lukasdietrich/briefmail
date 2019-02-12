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
	"os"
	"sync"
	"time"
)

type certFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

type TLS struct {
	Pem string
	Key string
}

func (t *TLS) MakeTLSConfig() *tls.Config {
	var f certFunc

	switch true {
	case t.Pem != "" && t.Key != "":
		f = certByFile(t.Pem, t.Key)

	default:
		return nil
	}

	return &tls.Config{GetCertificate: f}
}

func certByFile(pem, key string) certFunc {
	var (
		files    = []string{pem, key}
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
		cert, err := tls.LoadX509KeyPair(pem, key)
		if err != nil {
			return nil, err
		}

		lastCert = &cert
		lastTime = time.Now()

		return lastCert, nil
	}
}
