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
	"bytes"
	"io"
)

type prepender struct {
	lines [][]byte
}

func newPrepender(initialCapacity int) *prepender {
	return &prepender{
		lines: make([][]byte, 0, initialCapacity),
	}
}

func (p *prepender) reset() {
	p.lines = p.lines[:0]
}

func (p *prepender) prepend(key, value string) {
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

	p.lines = append(p.lines, buffer.Bytes())
}

func (p *prepender) reader(r io.Reader) io.Reader {
	readers := make([]io.Reader, 0, len(p.lines)+1)
	for _, line := range p.lines {
		readers = append(readers, bytes.NewReader(line))
	}

	return io.MultiReader(append(readers, r)...)
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
