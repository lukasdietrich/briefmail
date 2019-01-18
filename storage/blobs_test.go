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

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/lukasdietrich/briefmail/model"
)

func TestBlobs(t *testing.T) {
	var (
		blobs = Blobs{fs: afero.NewMemMapFs()}
		data  = make([]byte, 2<<16)

		id model.ID
		n  int64
	)

	rand.Seed(23980234)
	_, err := rand.Read(data)
	assert.Nil(t, err)

	t.Run("ReadNil", func(t *testing.T) {
		_, err := blobs.Read(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("Write", func(t *testing.T) {
		id, n, err = blobs.Write(bytes.NewReader(data))
		assert.Nil(t, err)
		assert.NotEqual(t, uuid.Nil, id)
		assert.EqualValues(t, n, len(data))
	})

	t.Run("Read", func(t *testing.T) {
		r, err := blobs.Read(id)
		assert.Nil(t, err)
		b, err := ioutil.ReadAll(r)
		assert.Nil(t, err)
		assert.Nil(t, r.Close())
		assert.Equal(t, data, b)
	})

	t.Run("ReadOffset", func(t *testing.T) {
		r, err := blobs.ReadOffset(id, 420)
		assert.Nil(t, err)
		b, err := ioutil.ReadAll(r)
		assert.Nil(t, err)
		assert.Nil(t, r.Close())
		assert.Equal(t, data[420:], b)
	})

	t.Run("Delete", func(t *testing.T) {
		assert.Nil(t, blobs.Delete(id))
		assert.Error(t, blobs.Delete(id))
	})
}
