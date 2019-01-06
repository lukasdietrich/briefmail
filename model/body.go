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

package model

import (
	"bytes"
	"io"
)

type Body struct {
	io.Reader
}

func (b *Body) Prepend(key, value string) {
	const (
		// see RFC#2822 2.1.1
		foldLength = 78
	)

	var (
		buffer bytes.Buffer

		length = len(key) + 2
		i      = 0
	)

	// allocate a buffer with enough space for the key and value plus
	// a little extra for folding line breaks
	buffer.Grow(len(key) + len(value) + 16)

	buffer.WriteString(key)
	buffer.WriteString(": ")

	for i < len(value) {
		foldPoint := findFoldPoint(value[i:], foldLength-length)

		buffer.WriteString(value[i : i+foldPoint])
		buffer.WriteString("\r\n")

		i += foldPoint
		length = 0
	}

	b.Reader = io.MultiReader(&buffer, b.Reader)
}

func findFoldPoint(line string, length int) int {
	const (
		space = 32
		tab   = 9
	)

	if len(line) > length {
		var candidate int

		for i, b := range line {
			if b == space || b == tab {
				candidate = i
			}

			if i >= length && candidate > 0 {
				return candidate
			}
		}
	}

	return len(line)
}
