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

package storage

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestBlobs(t *testing.T) {
	var (
		blobs = Blobs{fs: afero.NewMemMapFs()}
		data  = make([]byte, 2<<16)

		id string
	)

	rand.Seed(23980234)
	n, err := rand.Read(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	t.Run("ReadNil", func(t *testing.T) {
		_, err := blobs.Reader("")
		assert.Error(t, err)
	})

	t.Run("Write", func(t *testing.T) {
		var n int64
		id, n, err = blobs.Write(bytes.NewReader(data))
		assert.NoError(t, err)
		assert.NotEqual(t, "", id)
		assert.EqualValues(t, len(data), n)
	})

	t.Run("Read", func(t *testing.T) {
		r, err := blobs.Reader(id)
		assert.NoError(t, err)

		b, err := ioutil.ReadAll(r)
		assert.NoError(t, err)
		assert.NoError(t, r.Close())
		assert.Equal(t, data, b)
	})

	t.Run("ReadOffset", func(t *testing.T) {
		r, err := blobs.OffsetReader(id, 420)
		assert.NoError(t, err)

		b, err := ioutil.ReadAll(r)
		assert.NoError(t, err)
		assert.NoError(t, r.Close())
		assert.Equal(t, data[420:], b)
	})

	t.Run("Delete", func(t *testing.T) {
		assert.NoError(t, blobs.Delete(id))
		assert.Error(t, blobs.Delete(id))
	})
}
