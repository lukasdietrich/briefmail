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

package models

import (
	"database/sql/driver"
	"errors"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/text/cases"
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

// ParseNormalized calls ParseUnicode and transforms the local-part using NormalizeLocalPart.
func ParseNormalized(raw string) (Address, error) {
	addr, err := ParseUnicode(raw)
	if err != nil {
		return addr, err
	}

	localPart := NormalizeLocalPart(addr.LocalPart())
	if localPart != addr.LocalPart() {
		addr.raw = localPart + "@" + addr.Domain()
		addr.at = len(localPart)
	}

	return addr, nil
}

// ParseUnicode calls Parse and transforms the domain part of the address using DomainToUnicode.
func ParseUnicode(raw string) (Address, error) {
	addr, err := Parse(raw)
	if err != nil {
		return addr, err
	}

	domain, err := DomainToUnicode(addr.Domain())
	if err != nil {
		return addr, err
	}

	if domain != addr.Domain() {
		addr.raw = addr.LocalPart() + "@" + domain
	}

	return addr, nil
}

// Parse splits an address at the "@" sign and checks for size limits.
func Parse(raw string) (Address, error) {
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

// Normalized returns a copy of a with a normalized local-part.
func (a Address) Normalized() Address {
	localPart := a.LocalPart()
	localPart = NormalizeLocalPart(localPart)

	return Address{
		raw: localPart + "@" + a.Domain(),
		at:  len(localPart),
	}
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

	v, err := Parse(s.(string))
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

// fold is a cases.Caser to fold unicode text. Folding is more or less "compatible" lowercase.
var fold = cases.Fold()

// NormalizeLocalPart applie several rules to make the local-part of addresses comparable. This may
// only be applied to local addresses. Outbound addresses may not be altered.
//
// 1) The local-part is case-folded so that "user" and "USER" are considered equal.
// 2) The local-part is normalized using NFKC so that equal looking runes are considered equal.
// 3) The local-part has the suffix trimmed. A suffix is everything after the first '+' rune.
func NormalizeLocalPart(localPart string) string {
	folded := fold.String(localPart)
	normalized := norm.NFKC.String(folded)

	suffixIndex := strings.IndexRune(normalized, '+')
	if suffixIndex < 0 {
		return normalized
	}

	return normalized[:suffixIndex]
}
