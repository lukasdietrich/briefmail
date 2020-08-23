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
	"strconv"
	"strings"

	"github.com/lukasdietrich/briefmail/internal/textproto"
)

// command represents a command-line of the form:
//
//     <name> <SP> [<arg> [<SP> <arg>]*] <CR> <LF>
type command struct {
	name string
	args [][]byte
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
		c.name = string(line)
		c.args = nil
	} else {
		c.name = string(line[:space])
		c.args = bytes.Fields(line[space+1:])
	}

	c.name = strings.ToLower(c.name)
}

func (c *command) parseIndexArg(arg int) (int, error) {
	if len(c.args) <= arg {
		return -1, errInvalidSyntax
	}

	n, err := strconv.ParseInt(string(c.args[arg]), 10, 64)
	if err != nil {
		return -1, errInvalidSyntax
	}

	return int(n - 1), nil
}
