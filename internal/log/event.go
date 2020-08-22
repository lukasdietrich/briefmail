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
	"os"

	"github.com/rs/zerolog"
)

// Logger is the global zerolog.Logger instance.
var Logger = zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()

// Debug starts a new log event with debug level.
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// DebugContext starts a new log event with debug level and appends fields defined in the context.
func DebugContext(ctx context.Context) *zerolog.Event {
	return appendContextFields(ctx, Debug())
}

// Info starts a new log event with info level.
func Info() *zerolog.Event {
	return Logger.Info()
}

// InfoContext starts a new log event with info level and appends fields defined in the context.
func InfoContext(ctx context.Context) *zerolog.Event {
	return appendContextFields(ctx, Info())
}

// Warn starts a new log event with warn level.
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// WarnContext starts a new log event with warn level and appends fields defined in the context.
func WarnContext(ctx context.Context) *zerolog.Event {
	return appendContextFields(ctx, Warn())
}

// Error starts a new log event with error level.
func Error() *zerolog.Event {
	return Logger.Error()
}

// ErrorContext starts a new log event with error level and appends fields defined in the context.
func ErrorContext(ctx context.Context) *zerolog.Event {
	return appendContextFields(ctx, Error())
}

// Fatal starts a new log event with fatal level.
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// FatalContext starts a new log event with fatal level and appends fields defined in the context.
func FatalContext(ctx context.Context) *zerolog.Event {
	return appendContextFields(ctx, Fatal())
}
