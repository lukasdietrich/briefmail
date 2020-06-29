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

package smtp

import (
	"errors"
	"io"
)

var (
	errReaderLimitReached = errors.New("smtp: reader limit reached")
)

type limitedReader struct {
	r io.Reader
	n int64
}

func (l *limitedReader) Read(b []byte) (int, error) {
	if l.n <= 0 {
		return 0, errReaderLimitReached
	}

	if int64(len(b)) > l.n {
		b = b[:l.n]
	}

	n, err := l.r.Read(b)
	l.n -= int64(n)
	return n, err
}
