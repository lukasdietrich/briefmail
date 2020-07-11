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

// +build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/lukasdietrich/briefmail/internal/addressbook"
	"github.com/lukasdietrich/briefmail/internal/certs"
	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/pop3"
	"github.com/lukasdietrich/briefmail/internal/smtp"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var wireSet = wire.NewSet(
	wire.Struct(new(startCommand), "*"),
	wire.Struct(new(shellCommand), "*"),

	certs.NewTLSConfig,

	storage.NewDB,
	storage.NewBlobs,
	storage.NewCache,
	storage.NewCleaner,

	hook.FromHooks,
	hook.DataHooks,

	smtp.New,
	pop3.New,

	wire.Struct(new(delivery.Mailman), "*"),
	wire.Struct(new(delivery.QueueWorker), "*"),

	addressbook.Parse,
)

func newStartCommand() (*startCommand, error) {
	panic(wire.Build(wireSet))
}

func newShellCommand() (*shellCommand, error) {
	panic(wire.Build(wireSet))
}
