// Copyright (C) 2020  Lukas Dietrich <lukas@lukasdietrich.com>
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

package log

import (
	"bytes"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type baseLogTestSuite struct {
	suite.Suite

	buffer bytes.Buffer
}

func (s *baseLogTestSuite) SetupTest() {
	Logger = zerolog.New(&s.buffer).Level(zerolog.TraceLevel)
	s.buffer.Reset()
}

func (s *baseLogTestSuite) assertMsg(expected string) {
	s.Assert().Equal(expected, s.buffer.String())
}
