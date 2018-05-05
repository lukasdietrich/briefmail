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

// Writer is a buffered writer.
type Writer interface {
	io.Writer

	// WriteString writes a string to the buffer.
	WriteString(string) error

	// Endline writes a <CR> <LF> sequence to the buffer.
	Endline() error

	// Flush writes the buffer to the underlying direct writer.
	Flush() error

	// DotWriter returns an io.WriteCloser, which encodes text into a
	// dot-encoded sequence of lines. Upon closing a final dot line is written.
	DotWriter() io.WriteCloser
}

type writer struct {
	buffer *bufio.Writer
}

func newWriter(w io.Writer) *writer {
	return &writer{
		buffer: bufio.NewWriter(w),
	}
}

func (w *writer) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

func (w *writer) WriteString(s string) error {
	_, err := w.buffer.WriteString(s)
	return err
}

func (w *writer) Endline() error {
	_, err := w.buffer.WriteString("\r\n")
	return err
}

func (w *writer) Flush() error {
	return w.buffer.Flush()
}

func (w *writer) DotWriter() io.WriteCloser {
	return &dotWriter{w: w.buffer}
}
