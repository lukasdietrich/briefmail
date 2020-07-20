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

package mails

import (
	"database/sql/driver"
	"errors"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
)

var (
	// ErrInvalidAddressFormat is used for addresses of zero length or without
	// an "@" sign.
	ErrInvalidAddressFormat = errors.New("address: invalid format")

	// ErrPathTooLong is used for addresses, that are too long or contain a path
	// that is too long according to RFC#5321.
	ErrPathTooLong = errors.New("address: path too long")

	// ZeroAddress is an invalid, zero value Address.
	ZeroAddress Address
)

// Address is a string of the form "local-part@domain".
type Address struct {
	raw string
	at  int
}

// ParseWithNormalizedDomain calls ParseAddress and transforms the domain part
// of the address using DomainToUnicode.
func ParseWithNormalizedDomain(raw string) (Address, error) {
	addr, err := ParseAddress(raw)
	if err != nil {
		return addr, err
	}

	domain, err := DomainToUnicode(addr.Domain())
	if err != nil {
		return addr, err
	}

	addr.raw = addr.LocalPart() + "@" + domain
	return addr, nil
}

// ParseAddress splits an address at the "@" sign and checks for size limits.
func ParseAddress(raw string) (Address, error) {
	if len(raw) == 0 {
		return ZeroAddress, ErrInvalidAddressFormat
	}

	at := strings.LastIndex(raw, "@")
	if at < 0 {
		return ZeroAddress, ErrInvalidAddressFormat
	}

	// see RFC#5321 4.5.3.1
	if at > 64 || len(raw)-at > 256 || len(raw) > 256 {
		return ZeroAddress, ErrPathTooLong
	}

	return Address{raw, at}, nil
}

// String returns the raw address provided to ParseAddress.
func (a Address) String() string {
	return a.raw
}

// LocalPart returns the part left of the "@" sign (exclusive).
func (a Address) LocalPart() string {
	return a.raw[:a.at]
}

// Domain return the part right of the "@" sign (exclusive).
func (a Address) Domain() string {
	return a.raw[a.at+1:]
}

// Scan implements the sql.Scanner interface.
func (a *Address) Scan(src interface{}) error {
	s, err := driver.String.ConvertValue(src)
	if err != nil {
		return err
	}

	v, err := ParseAddress(s.(string))
	if err != nil {
		return err
	}

	*a = v
	return nil
}

// Value implements the sql/driver.Valuer interface.
func (a Address) Value() (driver.Value, error) {
	return a.raw, nil
}

// DomainToUnicode normalizes a punycode domain to unicode and applies the
// NFC normal form.
func DomainToUnicode(domain string) (string, error) {
	mapped, err := idna.Lookup.ToUnicode(domain)
	if err != nil {
		return domain, err
	}

	return norm.NFC.String(mapped), nil
}

// DomainToASCII transforms a unicode domain to punycode.
func DomainToASCII(domain string) (string, error) {
	mapped, err := DomainToUnicode(domain)
	if err != nil {
		return domain, err
	}

	return idna.Lookup.ToASCII(mapped)
}
