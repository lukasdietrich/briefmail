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
	"strconv"

	"github.com/lukasdietrich/briefmail/internal/textproto"
)

type reply struct {
	code int
	text string
}

func (r *reply) writeTo(w textproto.Writer) error {
	w.WriteString(strconv.Itoa(r.code))
	w.WriteString(" ")
	w.WriteString(r.text)
	w.Endline()

	return w.Flush()
}
