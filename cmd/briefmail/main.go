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
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/lukasdietrich/briefmail/storage"
)

var (
	Version string
)

var (
	DB    *storage.DB
	Blobs *storage.Blobs
	Cache *storage.Cache
)

func init() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	app := cli.NewApp()

	app.Name = "briefmail"
	app.Usage = "Briefly set up email"

	app.Version = Version

	app.Commands = []cli.Command{
		start(),
		mailbox(),
	}

	app.Before = before
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "enable verbose logging",
		},
		cli.StringFlag{
			Name:   "data",
			Usage:  "path to the data folder",
			EnvVar: "BRIEFMAIL_DATA",
			Value:  "./data/",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func before(ctx *cli.Context) error {
	if ctx.Bool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	err := os.MkdirAll(ctx.String("data"), 0700)
	if err != nil {
		return err
	}

	DB, err = storage.NewDB(filepath.Join(ctx.String("data"), "db.sqlite"))
	if err != nil {
		return err
	}

	Blobs, err = storage.NewBlobs(filepath.Join(ctx.String("data"), "blobs"))
	if err != nil {
		return err
	}

	Cache, err = storage.NewCache(
		filepath.Join(ctx.String("data"), "temp"),
		1048576, // 1 MiB
	)
	if err != nil {
		return err
	}

	return nil
}
