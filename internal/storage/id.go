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

package storage

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

var random = rand.Reader

// newRandomID generates a random id using the global random variable.
func newRandomID() (string, error) {
	const byteLength = 16

	b := make([]byte, byteLength)
	if _, err := io.ReadFull(random, b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
