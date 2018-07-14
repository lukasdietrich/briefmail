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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	var cmd command

	cmd.parse([]byte("NOOP"))

	assert.EqualValues(t, "NOOP", cmd.head)
	assert.Nil(t, cmd.tail)

	cmd.parse([]byte("VRFY foo@bar.com"))

	assert.EqualValues(t, "VRFY", cmd.head)
	assert.EqualValues(t, "foo@bar.com", cmd.tail)
}

func TestArg(t *testing.T) {
	var cmd command

	cmd.parse([]byte("MAIL FROM:<foo@bar.com> a=b c=d"))

	assert.EqualValues(t, "MAIL", cmd.head)
	assert.EqualValues(t, "FROM:<foo@bar.com> a=b c=d", cmd.tail)

	arg, params, err := cmd.args("FROM")

	assert.Nil(t, err)
	assert.EqualValues(t, "foo@bar.com", arg)
	assert.EqualValues(t, [][]byte{[]byte("a=b"), []byte("c=d")}, params)
}
