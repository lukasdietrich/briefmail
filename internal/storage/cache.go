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

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/log"
)

func init() {
	viper.SetDefault("storage.cache.foldername", "data/cache")
	viper.SetDefault("storage.cache.memorylimit", "20mb")
}

// CacheOptions are the configuration properties for the cache.
type CacheOptions struct {
	// Foldername is the folder to store temporary files in.
	Foldername string
	// MemoryLimit is the maximum size of files in bytes to be kept in memory.
	MemoryLimit uint
}

// CacheOptionsFromViper fills CacheOptions using viper.
func CacheOptionsFromViper() CacheOptions {
	return CacheOptions{
		Foldername:  viper.GetString("storage.cache.foldername"),
		MemoryLimit: viper.GetSizeInBytes("storage.cache.memorylimit"),
	}
}

// Cache is a temporary storage for blobs of data.
type Cache interface {
	// Write copies all data from the reader into temporary storage. If the total size exceeds the
	// configured limit, the data will be written to disk.
	Write(context.Context, io.Reader) (CacheEntry, error)
}

// CacheEntry is a single blob of data kept in temporary storage.
type CacheEntry interface {
	// Release deletes data on disk, that may have been written. If the size of the cache entry is
	// smaller than the memory limit, this is a noop.
	Release(context.Context) error
	// Reader returns a new reader to the full blob of data. This essentially seeks the start of the
	// file and is therefore not safe for concurrent use.
	Reader() (io.Reader, error)
}

// NewCache creates a new cache.
func NewCache(fs afero.Fs, idGen crypto.IDGenerator, opts CacheOptions) (Cache, error) {
	if err := fs.MkdirAll(opts.Foldername, 0700); err != nil {
		return nil, err
	}

	return &cache{
		fs:          afero.NewBasePathFs(fs, opts.Foldername),
		idGen:       idGen,
		memoryLimit: int64(opts.MemoryLimit),
	}, nil
}

type cache struct {
	fs          afero.Fs
	idGen       crypto.IDGenerator
	memoryLimit int64
}

func (c *cache) Write(ctx context.Context, r io.Reader) (CacheEntry, error) {
	memory := bytes.NewBuffer(nil)

	n, err := io.Copy(memory, io.LimitReader(r, c.memoryLimit))
	if err != nil {
		return nil, err
	}

	if n < c.memoryLimit {
		return memoryEntry(memory.Bytes()), nil
	}

	return c.evadeToFile(ctx, io.MultiReader(memory, r))
}

func (c *cache) evadeToFile(ctx context.Context, r io.Reader) (CacheEntry, error) {
	id, err := c.idGen.GenerateID()
	if err != nil {
		return nil, err
	}

	file, err := c.fs.Create(id)
	if err != nil {
		return nil, err
	}

	log.InfoContext(ctx).
		Str("filename", id).
		Int64("memoryLimit", c.memoryLimit).
		Msg("cache entry exceeding size limit, evading to file")

	if _, err := io.Copy(file, r); err != nil {
		log.WarnContext(ctx).
			Str("filename", id).
			Msg("could not write to cache file")

		if err := file.Close(); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not close partial cache file")
		}

		if err := c.fs.Remove(id); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not remove partial cache file")
		}

		return nil, err
	}

	return fileEntry{
		id:   id,
		file: file,
		fs:   c.fs,
	}, nil
}

type memoryEntry []byte

func (memoryEntry) Release(context.Context) error {
	return nil
}

func (m memoryEntry) Reader() (io.Reader, error) {
	return bytes.NewReader(m), nil
}

type fileEntry struct {
	id   string
	file afero.File
	fs   afero.Fs
}

func (f fileEntry) Release(ctx context.Context) error {
	log.InfoContext(ctx).
		Str("filename", f.id).
		Msg("removing cache file")

	if err := f.file.Close(); err != nil {
		return err
	}

	return f.fs.Remove(f.id)
}

func (f fileEntry) Reader() (io.Reader, error) {
	if _, err := f.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return f.file, nil
}
