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

package pop3

import (
	"bytes"

	"github.com/lukasdietrich/briefmail/internal/textproto"
)

// command represents a command-line of the form:
//
//     <head> <SP> <tail> <CR> <LF>
type command struct {
	head []byte
	tail []byte
}

func (c *command) readFrom(r textproto.Reader) error {
	line, err := r.ReadLine()
	if err != nil {
		return err
	}

	c.parse(line)
	return nil
}

func (c *command) parse(line []byte) {
	space := bytes.IndexRune(line, ' ')

	if space < 0 {
		c.head = line
		c.tail = nil
	} else {
		c.head = line[:space]
		c.tail = line[space+1:]
	}
}

func (c *command) args() [][]byte {
	if len(c.tail) == 0 {
		return nil
	}

	return bytes.Fields(c.tail)
}
