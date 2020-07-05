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

	storage.WireSet,
	certs.WireSet,
	hook.WireSet,
	smtp.WireSet,
	pop3.WireSet,
	delivery.WireSet,
	addressbook.WireSet,
)

func newStartCommand() (*startCommand, error) {
	panic(wire.Build(wireSet))
}

func newShellCommand() (*shellCommand, error) {
	panic(wire.Build(wireSet))
}
