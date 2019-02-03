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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilAddress(t *testing.T) {
	addr, err := ParseAddress("")
	assert.Nil(t, err)
	assert.Equal(t, NilAddress, addr)
}

func TestAddress(t *testing.T) {
	for raw, expected := range map[string]*Address{
		"user1@host1": {User: "user1", Domain: "host1"},
		"@example":    {User: "", Domain: "example"},
		"someone@":    {User: "someone", Domain: ""},
		"someone":     nil,

		fmt.Sprintf("%s@%s", longString(65), "example"):       nil,
		fmt.Sprintf("@%s", longString(256)):                   nil,
		fmt.Sprintf("%s@%s", longString(64), longString(255)): nil,
	} {
		t.Run(raw, func(t *testing.T) {
			actual, err := ParseAddress(raw)

			if expected != nil {
				assert.Nil(t, err)
				assert.Equal(t, expected.User, actual.User)
				assert.Equal(t, expected.Domain, actual.Domain)
				assert.Equal(t, raw, actual.String())
			} else {
				assert.Error(t, err)
				assert.Nil(t, actual)
			}
		})
	}
}

func longString(n int) string {
	r := make([]rune, n)
	for i := 0; i < n; i++ {
		r[i] = 'a'
	}

	return string(r)
}
