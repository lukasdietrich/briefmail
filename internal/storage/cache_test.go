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

func TestCache(t *testing.T) {
	cache := Cache{
		fs:          afero.NewMemMapFs(),
		memoryLimit: 1024,
	}

	data := make([]byte, 2048)

	rand.Seed(82347232)
	n, err := rand.Read(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	t.Run("InMemory", func(t *testing.T) {
		entry, err := cache.Write(bytes.NewReader(data[:1023]))
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.NotNil(t, entry.memory)
		assert.Nil(t, entry.file)

		for i := 0; i < 3; i++ {
			r, err := entry.Reader()
			assert.NoError(t, err)
			assert.NotNil(t, r)

			b, err := ioutil.ReadAll(r)
			assert.NoError(t, err)
			assert.Equal(t, data[:1023], b)
		}

		assert.NoError(t, entry.Release())
	})

	t.Run("OnDisk", func(t *testing.T) {
		entry, err := cache.Write(bytes.NewReader(data))
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Nil(t, entry.memory)
		assert.NotNil(t, entry.file)

		_, err = cache.fs.Stat(entry.file.Name())
		assert.NoError(t, err)

		for i := 0; i < 3; i++ {
			r, err := entry.Reader()
			assert.NoError(t, err)
			assert.NotNil(t, r)

			b, err := ioutil.ReadAll(r)
			assert.NoError(t, err)
			assert.Equal(t, data, b)
		}

		assert.Nil(t, entry.Release())

		_, err = cache.fs.Stat(entry.file.Name())
		assert.Error(t, err)
	})

}
