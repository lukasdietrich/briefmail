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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFold(t *testing.T) {
	msg := strings.Join([]string{
		"Subject: Very important mail",
		"",
		"This is important!",
	}, "\r\n")

	expected := strings.Join([]string{
		"Received: by very.good.mail.server (briefmail) for",
		" <a-very-important-person@very.good.mail.server>; Sat, 5 Jan 2019 06:33:36",
		" +0000 (UTC)",
		"Subject: Very important mail",
		"",
		"This is important!",
	}, "\r\n")

	p := newPrepender(1)
	p.prepend(
		"Received",
		"by very.good.mail.server (briefmail) "+
			"for <a-very-important-person@very.good.mail.server>"+
			"; Sat, 5 Jan 2019 06:33:36 +0000 (UTC)")

	var actual bytes.Buffer
	actual.ReadFrom(p.reader(strings.NewReader(msg)))

	assert.Equal(t, expected, actual.String())
}
