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
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomIDSource(t *testing.T) {
	random = rand.New(rand.NewSource(1337))

	id1, err := newRandomID()
	require.NoError(t, err)
	assert.Equal(t, "26c5a4182a817a42f545cbc6b1cd94a4", id1)
}

func TestRandomIDUnique(t *testing.T) {
	set := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id, err := newRandomID()
		require.NoError(t, err)
		assert.False(t, set[id])

		set[id] = true
	}
}
