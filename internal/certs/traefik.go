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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("tls.traefik.acme", "/etc/traefik/acme.json")
	viper.SetDefault("tls.traefik.domain", "localhost")
}

type traefikCertSource struct {
	acmeFilename string
	domain       string
}

func newTraefikCertSource() *traefikCertSource {
	return &traefikCertSource{
		acmeFilename: viper.GetString("tls.traefik.acme"),
		domain:       viper.GetString("tls.traefik.domain"),
	}
}

func (s *traefikCertSource) lastUpdate() (time.Time, error) {
	info, err := os.Stat(s.acmeFilename)
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

	f, err := os.Open(s.acmeFilename)
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

	return nil, fmt.Errorf("no certificate for domain=%s found in %s",
		s.domain, s.acmeFilename)
}
