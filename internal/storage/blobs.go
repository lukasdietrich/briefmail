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
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("storage.blobs.foldername", "data/blobs")
}

// Blobs is a permanent storage for blobs of data.
type Blobs struct {
	fs afero.Fs
}

// NewBlobs creates a new blobs store using configuration from viper.
//
// `storage.blobs.foldername` is the foldername for blob files.
func NewBlobs() (*Blobs, error) {
	folderName := viper.GetString("storage.blobs.foldername")

	if err := os.MkdirAll(folderName, 0700); err != nil {
		return nil, err
	}

	return &Blobs{
		fs: afero.NewBasePathFs(afero.NewOsFs(), folderName),
	}, nil
}

// Write copies all the data from r to a blob, that is addressable by the
// returned uuid.
func (b *Blobs) Write(r io.Reader) (uuid.UUID, int64, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return uuid.Nil, -1, err
	}

	f, err := b.fs.Create(id.String())
	if err != nil {
		return uuid.Nil, -1, err
	}

	logrus.Debugf("writing blob %s", id)

	size, err := io.Copy(f, r)
	if err != nil {
		f.Close()
		b.Delete(id)

		return uuid.Nil, -1, err
	}

	return id, size, f.Close()
}

// Delete removes a blob by id.
func (b *Blobs) Delete(id uuid.UUID) error {
	logrus.Debugf("removing blob %s", id)
	return b.fs.Remove(id.String())
}

// OffsetReader returns a reader to a blob with an initial offset to be skipped.
// The responsibiltiy to close the reader is on the caller.
func (b *Blobs) OffsetReader(id uuid.UUID, offset int64) (io.ReadCloser, error) {
	f, err := b.fs.Open(id.String())
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		_, err = f.Seek(offset, io.SeekStart)
	}

	return f, err
}

// Reader is a shorthand for OffsetReader(id, 0)
func (b *Blobs) Reader(id uuid.UUID) (io.ReadCloser, error) {
	return b.OffsetReader(id, 0)
}
