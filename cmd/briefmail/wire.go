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

	"github.com/lukasdietrich/briefmail/internal/certs"
	"github.com/lukasdietrich/briefmail/internal/delivery"
	"github.com/lukasdietrich/briefmail/internal/pop3"
	"github.com/lukasdietrich/briefmail/internal/smtp"
	"github.com/lukasdietrich/briefmail/internal/smtp/hook"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

func newStartCommand() (*startCommand, error) {
	panic(wire.Build(
		wire.Struct(new(startCommand), "*"),

		certs.NewTLSConfig,

		storage.OpenDatabase,
		storage.NewBlobs,
		storage.NewCache,

		hook.FromHooks,
		hook.DataHooks,

		smtp.New,
		pop3.New,

		delivery.NewAuthenticator,
		delivery.NewAddressbook,
		delivery.NewMailman,
		delivery.NewQueue,
		delivery.NewCourier,
		delivery.NewInboxer,
		delivery.NewCleaner,
	))
}

func newShellCommand() (*shellCommand, error) {
	panic(wire.Build(
		wire.Struct(new(shellCommand), "*"),

		storage.OpenDatabase,
	))
}
