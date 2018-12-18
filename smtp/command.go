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

package smtp

import (
	"bytes"
	"errors"

	"github.com/lukasdietrich/briefmail/textproto"
)

var (
	errCommandSyntax = errors.New("command: invalid syntax")
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

// parse a line into head and tail of a command.
// tail will be nil if no space is found.
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

func (c *command) args(name string) (arg string, params [][]byte, err error) {
	tail := c.tail

	if name != "" {
		if len(tail) < len(name)+3 { // len(FROM:<...>) < len(FROM) + len(:<>)
			err = errCommandSyntax
			return
		}

		if !bytes.HasPrefix(bytes.ToUpper(tail[:len(name)]), bytes.ToUpper([]byte(name))) {
			err = errCommandSyntax
			return
		}

		if !bytes.HasPrefix(tail[len(name):], []byte(":<")) {
			err = errCommandSyntax
			return
		}

		end := bytes.IndexRune(tail, '>')
		if end < 0 {
			err = errCommandSyntax
			return
		}

		arg = string(tail[len(name)+2 : end])
		tail = tail[end+1:]
	}

	if len(tail) > 0 {
		if tail[0] == ' ' {
			tail = tail[1:]
		}

		params = bytes.Fields(tail)
	}

	return
}
