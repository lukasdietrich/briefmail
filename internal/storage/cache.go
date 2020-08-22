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
	"context"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
)

func init() {
	viper.SetDefault("storage.cache.foldername", "data/cache")
	viper.SetDefault("storage.cache.memoryLimit", 1<<20) // 1 Megabyte
}

// Cache is a temporary storage for blobs of data.
type Cache struct {
	fs          afero.Fs
	memoryLimit int64
}

// NewCache creates a new cache using configuration from viper.
//
// `storage.cache.memoryLimit` is the maximum size of data written in memory.
// `storage.cache.foldername` is the foldername of temporary files.
func NewCache() (*Cache, error) {
	var (
		folderName  = viper.GetString("storage.cache.foldername")
		memoryLimit = viper.GetInt64("storage.cache.memoryLimit")
	)

	if err := os.MkdirAll(folderName, 0700); err != nil {
		return nil, err
	}

	return &Cache{
		fs:          afero.NewBasePathFs(afero.NewOsFs(), folderName),
		memoryLimit: memoryLimit,
	}, nil
}

// Write copies all the data from r into temporary storage. If the total size
// exceeds the configured limit, the data will be written to disk.
func (b *Cache) Write(ctx context.Context, r io.Reader) (*CacheEntry, error) {
	memory := bytes.NewBuffer(nil)

	n, err := io.Copy(memory, io.LimitReader(r, b.memoryLimit))
	if err != nil {
		return nil, err
	}

	if n < b.memoryLimit {
		return &CacheEntry{memory: memory}, nil
	}

	id, err := newRandomID()
	if err != nil {
		return nil, err
	}

	file, err := b.fs.Create(id)
	if err != nil {
		return nil, err
	}

	log.InfoContext(ctx).
		Str("filename", id).
		Int64("memoryLimit", b.memoryLimit).
		Msg("cache entry exceeding size limit, evading to file")

	if _, err := io.Copy(file, io.MultiReader(memory, r)); err != nil {
		log.WarnContext(ctx).
			Str("filename", id).
			Msg("could not write to cache file")

		if err := file.Close(); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not close partial cache file")
		}

		if err := b.fs.Remove(id); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not remove partial cache file")
		}

		return nil, err
	}

	return &CacheEntry{id: id, file: file, fs: b.fs}, nil
}

// CacheEntry is a single blob of data kept in temporary storage.
type CacheEntry struct {
	memory *bytes.Buffer
	id     string
	file   afero.File
	fs     afero.Fs
}

// Release deletes data on disk, that may have been written. If the size of the
// cache entry is smaller than the memory limit, this is a noop.
func (e *CacheEntry) Release(ctx context.Context) error {
	if e.file != nil {
		log.InfoContext(ctx).
			Str("filename", e.id).
			Msg("removing cache file")

		if err := e.file.Close(); err != nil {
			return err
		}

		return e.fs.Remove(e.id)
	}

	return nil
}

// Reader returns a new reader to the full blob of data. This essentially seeks
// the start of the file and is therefore not safe for concurrent use.
func (e *CacheEntry) Reader() (io.Reader, error) {
	if e.file != nil {
		if _, err := e.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		return e.file, nil
	}

	return bytes.NewReader(e.memory.Bytes()), nil
}
