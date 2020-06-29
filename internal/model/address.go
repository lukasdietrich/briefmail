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
	"database/sql/driver"
	"errors"
	"strings"

	"github.com/lukasdietrich/briefmail/internal/normalize"
)

var (
	ErrInvalidAddressFormat = errors.New("address: invalid format")
	ErrPathTooLong          = errors.New("address: path too long")

	NilAddress = &Address{
		raw:    "",
		User:   "",
		Domain: "",
	}
)

type Address struct {
	raw    string
	User   string
	Domain string
}

func ParseAddress(raw string) (*Address, error) {
	if len(raw) == 0 {
		return NilAddress, nil
	}

	if i := strings.LastIndex(raw, "@"); i > -1 {
		user, err := normalize.User(raw[:i])
		if err != nil {
			return nil, err
		}

		domain, err := normalize.Domain(raw[i+1:])
		if err != nil {
			return nil, err
		}

		addr := Address{
			raw:    raw,
			User:   user,
			Domain: domain,
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

func (a *Address) Value() (driver.Value, error) {
	return a.raw, nil
}

func (a *Address) Scan(v interface{}) error {
	v, err := driver.String.ConvertValue(v)
	if err != nil {
		return err
	}

	b, err := ParseAddress(v.(string))
	if err != nil {
		return err
	}

	*a = *b
	return nil
}

func (a *Address) MarshalText() ([]byte, error) {
	return []byte(a.raw), nil
}

func (a *Address) UnmarshalText(v []byte) error {
	b, err := ParseAddress(string(v))
	if err != nil {
		return err
	}

	*a = *b
	return nil
}
