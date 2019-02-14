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
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type Cache struct {
	fs          afero.Fs
	memoryLimit int64
}

func NewMemoryCache() (*Cache, error) {
	return &Cache{
		fs: afero.NewMemMapFs(),
	}, nil
}

func NewCache(folderName string, memoryLimit int64) (*Cache, error) {
	if err := os.MkdirAll(folderName, 0700); err != nil {
		return nil, err
	}

	return &Cache{
		fs:          afero.NewBasePathFs(afero.NewOsFs(), folderName),
		memoryLimit: memoryLimit,
	}, nil
}

func (b *Cache) Write(r io.Reader) (*CacheEntry, error) {
	memory := bytes.NewBuffer(nil)

	n, err := io.Copy(memory, io.LimitReader(r, b.memoryLimit))
	if err != nil {
		return nil, err
	}

	if n < b.memoryLimit {
		return &CacheEntry{
			memory: memory,
		}, nil
	}

	file, err := b.fs.Create(uuid.New().String())
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(file, io.MultiReader(memory, r)); err != nil {
		file.Close()             // nolint:errcheck
		b.fs.Remove(file.Name()) // nolint:errcheck
		return nil, err
	}

	return &CacheEntry{
		file: file,
		fs:   b.fs,
	}, nil
}

type CacheEntry struct {
	memory *bytes.Buffer
	file   afero.File
	fs     afero.Fs
}

func (e *CacheEntry) Release() error {
	if e.file != nil {
		if err := e.file.Close(); err != nil {
			return err
		}

		return e.fs.Remove(e.file.Name())
	}

	return nil
}

func (e *CacheEntry) Reader() (io.Reader, error) {
	if e.file != nil {
		if _, err := e.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		return e.file, nil
	}

	return bytes.NewReader(e.memory.Bytes()), nil
}
