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
	"github.com/spf13/afero"

	"github.com/lukasdietrich/briefmail/model"
)

type Blobs struct {
	fs afero.Fs
}

func NewBlobs(folderName string) (*Blobs, error) {
	if err := os.MkdirAll(folderName, 0700); err != nil {
		return nil, err
	}

	return &Blobs{
		fs: afero.NewBasePathFs(afero.NewOsFs(), folderName),
	}, nil
}

func (b *Blobs) Write(r io.Reader) (model.ID, int64, error) {
	id := model.NewID()

	f, err := b.fs.Create(id.String())
	if err != nil {
		return uuid.Nil, -1, err
	}

	size, err := io.Copy(f, r)
	if err != nil {
		f.Close()    // nolint:errcheck
		b.Delete(id) // nolint:errcheck

		return uuid.Nil, -1, err
	}

	return id, size, f.Close()
}

func (b *Blobs) Delete(id model.ID) error {
	return b.fs.Remove(id.String())
}

func (b *Blobs) ReadOffset(id model.ID, offset int64) (io.ReadCloser, error) {
	f, err := b.fs.Open(id.String())
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		_, err = f.Seek(offset, io.SeekStart)
	}

	return f, err
}

func (b *Blobs) Read(id model.ID) (io.ReadCloser, error) {
	return b.ReadOffset(id, 0)
}
