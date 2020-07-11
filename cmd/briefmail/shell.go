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
	"database/sql"
	"errors"
	"fmt"

	"github.com/abiosoft/ishell"

	"github.com/lukasdietrich/briefmail/internal/storage"
)

type shellCommand struct {
	DB *storage.DB
}

func (s *shellCommand) run() error {
	shell := ishell.New()
	s.setupShell(shell)
	shell.Run()

	return nil
}

func (s *shellCommand) setupShell(shell *ishell.Shell) {
	mailbox := ishell.Cmd{
		Name: "mailbox",
		Help: "manage mailboxes",
	}

	mailbox.AddCmd(&ishell.Cmd{
		Name: "add",
		Help: "add a new mailbox",
		Func: wrapShellFunc(s.addMailbox),
	})

	mailbox.AddCmd(&ishell.Cmd{
		Name: "passwd",
		Help: "update a mailbox password",
		Func: wrapShellFunc(s.changeMailboxPassword),
	})

	shell.AddCmd(&mailbox)
}

func (s *shellCommand) addMailbox(ctx *ishell.Context) error {
	if len(ctx.Args) != 1 {
		return errors.New("Usage: mailbox add [name]")
	}

	_, err := s.DB.Mailbox(ctx.Args[0])
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		return errors.New("name already taken")
	}

	ctx.Print("Password: ")
	pass, err := ctx.ReadPasswordErr()
	if err != nil {
		return err
	}

	if len(pass) == 0 {
		return errors.New("password must not be empty")
	}

	id, err := s.DB.AddMailbox(ctx.Args[0], pass)
	if err != nil {
		return fmt.Errorf("could not add mailbox: %w", err)
	}

	ctx.Printf("mailbox added with id=%d\n", id)
	return nil
}

func (s *shellCommand) changeMailboxPassword(ctx *ishell.Context) error {
	if len(ctx.Args) != 1 {
		return errors.New("Usage: mailbox passwd [name]")
	}

	_, err := s.DB.Mailbox(ctx.Args[0])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("mailbox does not exist")
		}

		return err
	}

	ctx.Print("Password: ")
	pass, err := ctx.ReadPasswordErr()
	if err != nil {
		return err
	}

	if len(pass) == 0 {
		return errors.New("password must not be empty")
	}

	if err := s.DB.SetPassword(ctx.Args[0], pass); err != nil {
		return fmt.Errorf("could not update mailbox password: %w", err)
	}

	ctx.Printf("password of %s changed\n", ctx.Args[0])
	return nil
}

func wrapShellFunc(fn func(*ishell.Context) error) func(*ishell.Context) {
	return func(ctx *ishell.Context) {
		if err := fn(ctx); err != nil {
			ctx.Err(err)
		}
	}
}
