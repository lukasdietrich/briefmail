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
	"os"
	"time"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("tls.files.crt", "cert/briefmail.crt")
	viper.SetDefault("tls.files.key", "cert/briefmail.key")
}

type filesCertSource struct {
	crtFilename string
	keyFilename string
}

func newFilesCertSource() *filesCertSource {
	return &filesCertSource{
		crtFilename: viper.GetString("tls.files.crt"),
		keyFilename: viper.GetString("tls.files.key"),
	}
}

func (s *filesCertSource) LastUpdate() (time.Time, error) {
	var updateTime time.Time

	for _, file := range [...]string{s.crtFilename, s.keyFilename} {
		info, err := os.Stat(file)
		if err != nil {
			return updateTime, err
		}

		if info.ModTime().After(updateTime) {
			updateTime = info.ModTime()
		}
	}

	return updateTime, nil
}

func (s *filesCertSource) Load() (*tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(s.crtFilename, s.keyFilename)
	if err != nil {
		return nil, err
	}

	return &certificate, nil
}
