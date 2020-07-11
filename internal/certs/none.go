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
	"errors"
	"time"
)

var (
	errCertSourceNone = errors.New("no certificate source is configured")
)

type noneCertSource struct{}

func (noneCertSource) lastUpdate() (time.Time, error) {
	return time.Time{}, errCertSourceNone
}

func (noneCertSource) load() (*tls.Certificate, error) {
	return nil, errCertSourceNone
}
