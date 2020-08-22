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

	"github.com/rs/zerolog"
)

type fieldConnection struct{}
type fieldOrigin struct{}
type fieldCommand struct{}

// WithConnection adds the connection identifier to the context.
func WithConnection(ctx context.Context, connection int32) context.Context {
	return context.WithValue(ctx, fieldConnection{}, connection)
}

// WithOrigin adds the origin of processing to the context.
func WithOrigin(ctx context.Context, origin string) context.Context {
	return context.WithValue(ctx, fieldOrigin{}, origin)
}

// WithCommand adds the command name to the context.
func WithCommand(ctx context.Context, command string) context.Context {
	return context.WithValue(ctx, fieldCommand{}, command)
}

// appendContextFields adds defined fields in the context to the log event.
func appendContextFields(ctx context.Context, event *zerolog.Event) *zerolog.Event {
	if connection, ok := ctx.Value(fieldConnection{}).(int32); ok {
		event.Int32("connection", connection)
	}

	if origin, ok := ctx.Value(fieldOrigin{}).(string); ok {
		event.Str("origin", origin)
	}

	if command, ok := ctx.Value(fieldCommand{}).(string); ok {
		event.Str("command", command)
	}

	return event
}
