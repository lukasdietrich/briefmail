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

func TestLogEventTestSuite(t *testing.T) {
	suite.Run(t, new(LogEventTestSuite))
}

type LogEventTestSuite struct {
	baseLogTestSuite
}

func (s *LogEventTestSuite) TestTrace() {
	Trace().Msg("TestTrace")
	s.assertMsg("{\"level\":\"trace\",\"message\":\"TestTrace\"}\n")
}

func (s *LogEventTestSuite) TestTraceContext() {
	TraceContext(WithOrigin(context.TODO(), "o1")).Msg("TestTraceContext")
	s.assertMsg("{\"level\":\"trace\",\"origin\":\"o1\",\"message\":\"TestTraceContext\"}\n")
}

func (s *LogEventTestSuite) TestDebug() {
	Debug().Msg("TestDebug")
	s.assertMsg("{\"level\":\"debug\",\"message\":\"TestDebug\"}\n")
}

func (s *LogEventTestSuite) TestDebugContext() {
	DebugContext(WithOrigin(context.TODO(), "o2")).Msg("TestDebugContext")
	s.assertMsg("{\"level\":\"debug\",\"origin\":\"o2\",\"message\":\"TestDebugContext\"}\n")
}

func (s *LogEventTestSuite) TestInfo() {
	Info().Msg("TestInfo")
	s.assertMsg("{\"level\":\"info\",\"message\":\"TestInfo\"}\n")
}

func (s *LogEventTestSuite) TestInfoContext() {
	InfoContext(WithOrigin(context.TODO(), "o3")).Msg("TestInfoContext")
	s.assertMsg("{\"level\":\"info\",\"origin\":\"o3\",\"message\":\"TestInfoContext\"}\n")
}

func (s *LogEventTestSuite) TestWarn() {
	Warn().Msg("TestWarn")
	s.assertMsg("{\"level\":\"warn\",\"message\":\"TestWarn\"}\n")
}

func (s *LogEventTestSuite) TestWarnContext() {
	WarnContext(WithOrigin(context.TODO(), "o4")).Msg("TestWarnContext")
	s.assertMsg("{\"level\":\"warn\",\"origin\":\"o4\",\"message\":\"TestWarnContext\"}\n")
}

func (s *LogEventTestSuite) TestError() {
	Error().Msg("TestError")
	s.assertMsg("{\"level\":\"error\",\"message\":\"TestError\"}\n")
}

func (s *LogEventTestSuite) TestErrorContext() {
	ErrorContext(WithOrigin(context.TODO(), "o5")).Msg("TestErrorContext")
	s.assertMsg("{\"level\":\"error\",\"origin\":\"o5\",\"message\":\"TestErrorContext\"}\n")
}
