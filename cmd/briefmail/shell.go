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

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/abiosoft/ishell"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

type shellCommand struct {
	Database *storage.Database
}

func (s *shellCommand) run() error {
	shell := ishell.New()
	s.setupShell(shell)
	shell.Run()

	return nil
}

func (s *shellCommand) setupShell(shell *ishell.Shell) {
	shell.AddCmd(composeShellCmd(
		ishell.Cmd{
			Name: "domains",
			Help: "manage domains",
		},
		[]*ishell.Cmd{
			{
				Name: "list",
				Help: "list all domains",
				Func: s.wrapShellFunc(s.domainsList),
			},
			{
				Name: "add",
				Help: "add a new domain",
				Func: s.wrapShellFunc(s.domainsAdd),
			},
			{
				Name: "remove",
				Help: "remove a domain",
				Func: s.wrapShellFunc(s.domainsRemove),
			},
		},
	))

	shell.AddCmd(composeShellCmd(
		ishell.Cmd{
			Name: "addresses",
			Help: "manage addresses",
		},
		[]*ishell.Cmd{
			{
				Name: "list",
				Help: "list all addresses",
				Func: s.wrapShellFunc(s.addressesList),
			},
			{
				Name: "add",
				Help: "add a new address with its own mailbox",
				Func: s.wrapShellFunc(s.addressesAdd),
			},
			{
				Name: "remove",
				Help: "remove an address",
				Func: s.wrapShellFunc(s.addressesRemove),
			},
		},
	))
}

func (s *shellCommand) domainsList(ctx shellContext) error {
	if !ctx.checkArgs(0) {
		return errors.New("Usage: domains list")
	}

	domains, err := queries.FindDomains(ctx.tx)
	if err != nil {
		return err
	}

	ctx.printf("\n(%d) Domains:\n", len(domains))
	for _, domain := range domains {
		ctx.printf("\t%q\n", domain.Name)
	}
	ctx.printf("\n")

	return nil
}

func (s *shellCommand) domainsAdd(ctx shellContext) error {
	if !ctx.checkArgs(1) {
		return errors.New("Usage: domains add [DOMAIN]")
	}

	name, err := mails.DomainToUnicode(ctx.arg(0))
	if err != nil {
		return err
	}

	domain := storage.Domain{
		Name: name,
	}

	if err := queries.InsertDomain(ctx.tx, &domain); err != nil {
		return err
	}

	ctx.printf("\n\tDomain %q added.\n\n", name)
	return nil
}

func (s *shellCommand) domainsRemove(ctx shellContext) error {
	if !ctx.checkArgs(1) {
		return errors.New("Usage: domains remove [DOMAIN]")
	}

	name, err := mails.DomainToUnicode(ctx.arg(0))
	if err != nil {
		return err
	}

	if err := queries.DeleteDomain(ctx.tx, name); err != nil {
		return err
	}

	ctx.printf("\n\tDomain %q deleted.\n\n", name)
	return nil
}

func (s *shellCommand) addressesList(ctx shellContext) error {
	if !ctx.checkArgs(1) {
		return errors.New("Usage: addresses list [DOMAIN]")
	}

	domain, err := mails.DomainToUnicode(ctx.arg(0))
	if err != nil {
		return err
	}

	ok, err := queries.ExistsDomain(ctx.tx, domain)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("domain %q does not exist", domain)
	}

	addresses, err := queries.FindAddressesByDomain(ctx.tx, domain)
	if err != nil {
		return err
	}

	ctx.printf("\n(%d) Addresses:\n", len(addresses))
	for _, address := range addresses {
		ctx.printf("\t%s@%s\n", address.LocalPart, domain)
	}
	ctx.printf("\n")

	return nil
}

func (s *shellCommand) addressesAdd(ctx shellContext) error {
	if !ctx.checkArgs(1) {
		return errors.New("Usage: addresses add [ADDRESS]")
	}

	addr, err := mails.ParseNormalized(ctx.arg(0))
	if err != nil {
		return err
	}

	domain, err := queries.FindDomain(ctx.tx, addr.Domain())
	if err != nil {
		return err
	}

	pass, err := ctx.ask("Password", true)
	if err != nil {
		return err
	}

	var mailbox storage.Mailbox
	if err := crypto.Hash(&mailbox, []byte(pass)); err != nil {
		return err
	}

	if err := queries.InsertMailbox(ctx.tx, &mailbox); err != nil {
		return err
	}

	address := storage.Address{
		LocalPart: addr.LocalPart(),
		DomainID:  domain.ID,
		MailboxID: mailbox.ID,
	}

	if err := queries.InsertAddress(ctx.tx, &address); err != nil {
		return err
	}

	ctx.printf("\nAddress %q added.\n\n", addr)
	return nil
}

func (s *shellCommand) addressesRemove(ctx shellContext) error {
	if !ctx.checkArgs(1) {
		return errors.New("Usage: addresses remove [ADDRESS]")
	}

	return errors.New("not yet implemented")
}

type shellContext struct {
	shell *ishell.Context
	tx    *storage.Tx
}

func (c *shellContext) checkArgs(n int) bool {
	return len(c.shell.Args) == n
}

func (c *shellContext) arg(i int) string {
	return c.shell.Args[i]
}

func (c *shellContext) printf(format string, v ...interface{}) {
	c.shell.Printf(format, v...)
}

func (c *shellContext) ask(prompt string, hide bool) (string, error) {
	c.printf("%s: ", prompt)

	if hide {
		return c.shell.ReadPasswordErr()
	}

	return c.shell.ReadLineErr()
}

func composeShellCmd(cmd ishell.Cmd, children []*ishell.Cmd) *ishell.Cmd {
	for _, child := range children {
		cmd.AddCmd(child)
	}

	return &cmd
}

func (s *shellCommand) wrapShellFunc(fn func(shellContext) error) func(*ishell.Context) {
	return func(shell *ishell.Context) {
		tx, err := s.Database.BeginTx(context.Background())
		if err != nil {
			shell.Err(err)
			return
		}

		defer tx.Rollback()

		ctx := shellContext{
			shell: shell,
			tx:    tx,
		}

		if err := fn(ctx); err != nil {
			shell.Err(err)
			return
		}

		if err := tx.Commit(); err != nil {
			shell.Err(err)
		}
	}
}
