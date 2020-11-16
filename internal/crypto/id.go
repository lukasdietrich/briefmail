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

package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// IDGenerator is a service to generate unique string IDs.
type IDGenerator interface {
	// GenerateID generates a new id.
	GenerateID() (string, error)
}

// NewIDGenerator creates a new id generator.
func NewIDGenerator() IDGenerator {
	return &randomIDGenerator{random: rand.Reader}
}

type randomIDGenerator struct {
	random io.Reader
}

func (r randomIDGenerator) GenerateID() (string, error) {
	const byteLength = 16

	b, err := r.readRandomBytes(byteLength)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (r randomIDGenerator) readRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(r.random, b)
	return b, err
}
