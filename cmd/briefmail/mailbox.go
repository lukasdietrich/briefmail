// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
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

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func mailbox() cli.Command {
	return cli.Command{
		Name:  "mailbox",
		Usage: "Manage mailboxes",
		Subcommands: []cli.Command{
			{
				Name:  "add",
				Usage: "Add a new mailbox",
				Action: func(ctx *cli.Context) error {
					name, err := promptMailbox(false)
					if err != nil {
						return err
					}

					pass, err := promptPassword()
					if err != nil {
						return err
					}

					id, err := DB.AddMailbox(name, pass)
					if err != nil {
						return err
					}

					logrus.Infof("mailbox added (id=%d)", id)
					return nil
				},
			},
			{
				Name:  "passwd",
				Usage: "Update a mailbox password",
				Action: func(ctx *cli.Context) error {
					name, err := promptMailbox(true)
					if err != nil {
						return err
					}

					pass, err := promptPassword()
					if err != nil {
						return err
					}

					if err := DB.SetPassword(name, pass); err != nil {
						return err
					}

					logrus.Infof("password updated")
					return nil
				},
			},
		},
	}
}

func promptMailbox(shouldExist bool) (string, error) {
	var name string

	return name, survey.AskOne(&survey.Input{Message: "Name:"}, &name,
		survey.ComposeValidators(survey.Required, func(v interface{}) error {
			switch _, err := DB.Mailbox(v.(string)); err {
			case nil:
				if !shouldExist {
					return errors.New("name already taken")
				}

			case sql.ErrNoRows:
				if shouldExist {
					return errors.New("no such mailbox")
				}

			default:
				return err
			}

			return nil
		}))

}

func promptPassword() (string, error) {
	var pass string

	return pass, survey.AskOne(&survey.Password{Message: "Password:"}, &pass,
		survey.Required)
}
