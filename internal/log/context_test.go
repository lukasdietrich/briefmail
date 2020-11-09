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
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestLogContextTestSuite(t *testing.T) {
	suite.Run(t, new(LogContextTestSuite))
}

type LogContextTestSuite struct {
	baseLogTestSuite
}

func (s *LogContextTestSuite) TestWithOrigin() {
	ctx := WithOrigin(context.TODO(), "origin1")
	InfoContext(ctx).Msg("TestWithOrigin")

	s.assertMsg("{\"level\":\"info\",\"origin\":\"origin1\",\"message\":\"TestWithOrigin\"}\n")
}

func (s *LogContextTestSuite) TestWithCommand() {
	ctx := WithCommand(context.TODO(), "cmd1")
	InfoContext(ctx).Msg("TestWithCommand")

	s.assertMsg("{\"level\":\"info\",\"command\":\"cmd1\",\"message\":\"TestWithCommand\"}\n")
}

func (s *LogContextTestSuite) TestWithConnection() {
	ctx := WithConnection(context.TODO(), 123)
	InfoContext(ctx).Msg("TestWithConnection")

	s.assertMsg("{\"level\":\"info\",\"connection\":123,\"message\":\"TestWithConnection\"}\n")
}

func (s *LogContextTestSuite) TestWithAll() {
	ctx := context.TODO()
	ctx = WithOrigin(ctx, "origin2")
	ctx = WithCommand(ctx, "cmd3")
	ctx = WithConnection(ctx, 456)
	InfoContext(ctx).Msg("TestWithAll")

	s.assertMsg("{\"level\":\"info\"," +
		"\"connection\":456,\"origin\":\"origin2\",\"command\":\"cmd3\"," +
		"\"message\":\"TestWithAll\"}\n")
}
