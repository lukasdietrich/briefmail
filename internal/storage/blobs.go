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
	"context"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/log"
)

func init() {
	viper.SetDefault("storage.blobs.foldername", "data/blobs")
}

// BlobsOptions are the configuration properties for the blob store.
type BlobsOptions struct {
	Foldername string
}

// BlobsOptionsFromViper fills BlobsOptions using viper.
func BlobsOptionsFromViper() BlobsOptions {
	return BlobsOptions{
		Foldername: viper.GetString("storage.blobs.foldername"),
	}
}

// Blobs is a persistent storage for blobs of data.
type Blobs interface {
	// Write copies all the data from r to a blob, that is addressable by the returned id.
	Write(ctx context.Context, r io.Reader) (id string, size int64, err error)
	// Delete removes a blob by id.
	Delete(ctx context.Context, id string) error
	// OffsetReader returns a reader to a blob with an initial offset to be skipped.
	// The responsibiltiy to close the reader is on the caller.
	OffsetReader(id string, offset int64) (io.ReadCloser, error)
	// Reader is a shorthand for OffsetReader(id, 0)
	Reader(id string) (io.ReadCloser, error)
}

// NewBlobs creates a new blob store.
func NewBlobs(fs afero.Fs, idGen crypto.IDGenerator, opts BlobsOptions) (Blobs, error) {
	if err := fs.MkdirAll(opts.Foldername, 0700); err != nil {
		return nil, err
	}

	return &blobs{
		fs:    afero.NewBasePathFs(fs, opts.Foldername),
		idGen: idGen,
	}, nil
}

type blobs struct {
	fs    afero.Fs
	idGen crypto.IDGenerator
}

func (b *blobs) Write(ctx context.Context, r io.Reader) (string, int64, error) {
	id, err := b.idGen.GenerateID()
	if err != nil {
		return id, -1, err
	}

	f, err := b.fs.Create(id)
	if err != nil {
		return id, -1, err
	}

	log.InfoContext(ctx).
		Str("filename", id).
		Msg("writing blob")

	size, err := io.Copy(f, r)
	if err != nil {
		log.WarnContext(ctx).
			Str("filename", id).
			Msg("could not write to blob file")

		if err := f.Close(); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not close partial blob file")
		}

		if err := b.fs.Remove(id); err != nil {
			log.WarnContext(ctx).
				Str("filename", id).
				Err(err).
				Msg("could not remove partial blob file")
		}

		return id, -1, err
	}

	return id, size, f.Close()
}

func (b *blobs) Delete(ctx context.Context, id string) error {
	log.InfoContext(ctx).
		Str("filename", id).
		Msg("removing blob")

	return b.fs.Remove(id)
}

func (b *blobs) OffsetReader(id string, offset int64) (io.ReadCloser, error) {
	if id == "" {
		return nil, os.ErrInvalid
	}

	f, err := b.fs.Open(id)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		_, err = f.Seek(offset, io.SeekStart)
	}

	return f, err
}

func (b *blobs) Reader(id string) (io.ReadCloser, error) {
	return b.OffsetReader(id, 0)
}
