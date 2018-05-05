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
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func joinLines(lines []string) string {
	// ensure final <CR> <LF>
	lines = append(lines, "")
	return strings.Join(lines, "\r\n")
}

func TestDotReader(t *testing.T) {
	sequence := []string{
		"first line",
		"normal line",
		".with a dot",
		"",
		"..",
		".",
		"last line",
	}

	decoded := []string{
		"normal line",
		"with a dot",
		"",
		".",
	}

	buffer := bytes.NewBufferString(joinLines(sequence))
	reader := newReader(buffer)

	{ // test before dot encoded block
		line, err := reader.ReadLine()
		assert.Nil(t, err)
		assert.EqualValues(t, sequence[0], line)
	}

	{ // test a handful of dot encoded lines
		text, err := ioutil.ReadAll(reader.DotReader())
		assert.Nil(t, err)
		assert.EqualValues(t, joinLines(decoded), text)
	}

	{ // resume after dot encoded block
		line, err := reader.ReadLine()
		assert.Nil(t, err)
		assert.EqualValues(t, sequence[6], line)
	}

}

func TestDotWriter(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	writer := newWriter(buffer)

	assert.Nil(t, writer.WriteString("first line"))
	assert.Nil(t, writer.Endline())

	{
		decoded := []string{
			"normal line",
			".with a dot",
			".",
			"",
			"another",
		}

		encoder := writer.DotWriter()
		io.Copy(encoder, bytes.NewBufferString(joinLines(decoded)))

		assert.Nil(t, encoder.Close())
		assert.Nil(t, writer.Flush())
	}

	expected := []string{
		"first line",
		"normal line",
		"..with a dot",
		"..",
		"",
		"another",
		".",
	}

	assert.EqualValues(t, joinLines(expected), buffer.Bytes())
}
