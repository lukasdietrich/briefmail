// Copyright (C) 2018  Lukas Dietrich <lukas@lukasdietrich.com>
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

package textproto

import (
	"bufio"
	"io"
)

// Reader is a buffer for line based reading.
type Reader interface {
	// ReadLine tries to read the next line, but may fail and return an error.
	// If the line is non-nil, the error is nil and vice versa.
	ReadLine() ([]byte, error)
	// DotReader returns an io.Reader, which decodes a dot-encoded block of text
	// up to, but discarding, the closing dot line.
	DotReader() io.Reader
}

type reader struct {
	buffer *bufio.Scanner
}

func newReader(r io.Reader) *reader {
	return &reader{
		buffer: bufio.NewScanner(r),
	}
}

func (r *reader) ReadLine() ([]byte, error) {
	if !r.buffer.Scan() {
		if err := r.buffer.Err(); err != nil {
			return nil, err
		}

		return nil, io.EOF
	}

	return r.buffer.Bytes(), nil
}

func (r *reader) DotReader() io.Reader {
	return &dotReader{r: r}
}
