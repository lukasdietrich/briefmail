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

package model

import (
	"errors"
	"strings"
)

var (
	ErrInvalidAddressFormat = errors.New("address: invalid format")
	ErrPathTooLong          = errors.New("address: path too long")
)

type Address struct {
	raw    string
	User   string
	Domain string
}

func ParseAddress(raw string) (*Address, error) {
	if i := strings.LastIndex(raw, "@"); i > -1 {
		addr := Address{
			raw:    raw,
			User:   raw[:i],
			Domain: strings.ToLower(raw[i+1:]),
		}

		// see RFC#5321 4.5.3.1
		if len(addr.User) > 64 || len(addr.Domain) > 255 || len(raw) > 256 {
			return nil, ErrPathTooLong
		}

		return &addr, nil
	}

	return nil, ErrInvalidAddressFormat
}

func (a *Address) String() string {
	return a.raw
}
