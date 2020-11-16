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
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIDSource(t *testing.T) {
	idGen := randomIDGenerator{random: rand.New(rand.NewSource(1337))}

	id, err := idGen.GenerateID()
	require.NoError(t, err)
	assert.Equal(t, "26c5a4182a817a42f545cbc6b1cd94a4", id)
}

func TestGenerateIDUnique(t *testing.T) {
	idGen := NewIDGenerator()
	set := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id, err := idGen.GenerateID()
		require.NoError(t, err)
		assert.False(t, set[id])

		set[id] = true
	}
}

func TestGenerateIDError(t *testing.T) {
	idGen := randomIDGenerator{random: strings.NewReader("too-short")}

	id, err := idGen.GenerateID()
	assert.Error(t, err)
	assert.Zero(t, id)
}
