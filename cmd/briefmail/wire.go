// +build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/lukasdietrich/briefmail/addressbook"
	"github.com/lukasdietrich/briefmail/certs"
	"github.com/lukasdietrich/briefmail/delivery"
	"github.com/lukasdietrich/briefmail/pop3"
	"github.com/lukasdietrich/briefmail/smtp"
	"github.com/lukasdietrich/briefmail/smtp/hook"
	"github.com/lukasdietrich/briefmail/storage"
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
