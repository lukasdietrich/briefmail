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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyAddress(t *testing.T) {
	addr, err := Parse("")
	assert.Equal(t, ErrInvalidAddressFormat, err)
	assert.Zero(t, addr)
}

func TestInvalidAddress(t *testing.T) {
	addr, err := Parse("no-at-sign")
	assert.Equal(t, ErrInvalidAddressFormat, err)
	assert.Zero(t, addr)
}

func TestTooLongAddress(t *testing.T) {
	for _, raw := range []string{
		longString(200) + "@" + longString(200),
		"@" + longString(256),
		longString(65) + "@",
		longString(64) + "@" + longString(192),
	} {
		addr, err := Parse(raw)
		assert.Equal(t, ErrPathTooLong, err)
		assert.Zero(t, addr)
	}
}

func TestValidAddress(t *testing.T) {
	for _, raw := range []string{
		longString(64) + "@" + longString(100),
		"@" + longString(255),
		longString(10) + "@" + longString(245),
	} {
		addr, err := Parse(raw)
		assert.NoError(t, err)
		assert.NotZero(t, addr)
		assert.Equal(t, raw, addr.String())
	}
}

func longString(n int) string {
	r := make([]rune, n)
	for i := 0; i < n; i++ {
		r[i] = 'a'
	}

	return string(r)
}

func TestDomainToASCII(t *testing.T) {
	for domain, expected := range map[string]string{
		"example.com":     "example.com",
		"dömäin.example":  "xn--dmin-moa0i.example",
		"DÖMÄIN.example":  "xn--dmin-moa0i.example",
		"äaaa.example":    "xn--aaa-pla.example",
		"déjà.vu.example": "xn--dj-kia8a.vu.example",
		"fußball.example": "fussball.example",
	} {
		actual, err := DomainToASCII(domain)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
}

func TestDomainToUnicode(t *testing.T) {
	for domain, expected := range map[string]string{
		"example.com":             "example.com",
		"xn--dmin-moa0i.example":  "dömäin.example",
		"xn--aaa-pla.example":     "äaaa.example",
		"xn--dj-kia8a.vu.example": "déjà.vu.example",
		"fussball.example":        "fussball.example",
	} {
		actual, err := DomainToUnicode(domain)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
}

func TestNormalizeLocalPart(t *testing.T) {
	for localPart, expected := range map[string]string{
		"user+suffix":                    "user",
		"fußball":                        "fussball",
		"ÄÖÜ":                            "äöü",
		"\u0041\u030A+and+a+long+suffix": "\u00e5",
	} {
		actual := NormalizeLocalPart(localPart)
		assert.Equal(t, expected, actual)
	}
}
