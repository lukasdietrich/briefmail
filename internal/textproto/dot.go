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

const (
	sStart int = iota
	sCr
	sText
	sEOF
)

type dotReader struct {
	r     *reader
	state int

	line []byte
	i    int
}

func (d *dotReader) readByte() (byte, error) {
	switch d.state {
	case sStart:
		line, err := d.r.ReadLine()
		if err != nil {
			return 0, err
		}

		if len(line) == 1 && line[0] == '.' {
			d.state = sEOF
			return 0, io.EOF
		}

		d.line = line
		d.i = 0
		d.state = sText

		if len(line) > 1 && line[0] == '.' {
			d.i++
		}

		fallthrough
	case sText:
		if d.i < len(d.line) {
			r := d.line[d.i]
			d.i++
			return r, nil
		}

		d.state = sCr
		return '\r', nil
	case sCr:
		d.state = sStart
		return '\n', nil
	}

	return 0, io.EOF
}

func (d *dotReader) Read(b []byte) (int, error) {
	var n int

	for n < len(b) {
		r, err := d.readByte()
		if err != nil {
			if err != io.EOF || n == 0 {
				return 0, err
			}

			break
		}

		b[n] = r
		n++
	}

	return n, nil
}

type dotWriter struct {
	w     *bufio.Writer
	state int
}

func (d *dotWriter) Write(b []byte) (int, error) {
	var (
		i int
		w = d.w
	)

	for i < len(b) {
		r := b[i]

		// nolint:errcheck
		switch d.state {
		case sStart:
			d.state = sText
			if r == '.' {
				w.WriteByte('.')
			}

			fallthrough
		case sText:
			switch r {
			case '\r':
				d.state = sCr
			case '\n':
				w.WriteByte('\r')
				d.state = sStart
			}
		case sCr:
			d.state = sText
			if r == '\n' {
				d.state = sStart
			}
		}

		if err := w.WriteByte(r); err != nil {
			return i, err
		}

		i++
	}

	return i, nil
}

func (d *dotWriter) Close() error {
	// nolint:errcheck
	if d.state != sStart {
		if d.state == sText {
			d.w.WriteByte('\r')
		}
		d.w.WriteByte('\n')
	}

	_, err := d.w.WriteString(".\r\n")
	return err
}
