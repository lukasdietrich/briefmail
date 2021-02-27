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

package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/jmoiron/sqlx"
	"github.com/lukasdietrich/groundwork"
	"github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
)

const (
	driverName     = "sqlite3"
	changelogTable = "database_changelog"
)

//go:embed changesets/*
var changesetFolder embed.FS

func init() {
	viper.SetDefault("storage.database.filename", "data/briefmail.sqlite")
	viper.SetDefault("storage.database.journalmode", "wal")
}

// Queryer is an interface for both transactions and the database connection itself.
type Queryer interface {
	sqlx.ExtContext
}

// Tx is a database transaction, which can be rolled back or committed.
type Tx interface {
	Queryer
	Commit() error
	Rollback() error
	RollbackWith(func()) error
}

type tx struct {
	*sqlx.Tx
}

func (t tx) RollbackWith(callback func()) error {
	err := t.Rollback()

	if !errors.Is(err, sql.ErrTxDone) {
		callback()
	}

	return err
}

// Conn is a connection to the sql database.
type Conn interface {
	Queryer
	Begin(context.Context) (Tx, error)
	Close() error
}

type conn struct {
	*sqlx.DB
}

func (c conn) Begin(ctx context.Context) (Tx, error) {
	rawTx, err := c.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return tx{rawTx}, nil
}

// OpenConnection opens an sqlite3 database connection using the configuration from viper.
func OpenConnection() (Conn, error) {
	sqliteVersion, _, _ := sqlite3.Version()

	dsn := createDataSourceName()
	log.Info().
		Str("driver", driverName).
		Str("version", sqliteVersion).
		Str("dataSourceName", dsn).
		Msg("connecting to database")

	db, err := sqlx.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	changesets, err := loadChangesets()
	if err != nil {
		return nil, err
	}

	dialect := groundwork.Sqlite(db.DB, changelogTable)

	if err := groundwork.Up(dialect, changesets); err != nil {
		return nil, err
	}

	return conn{db}, nil
}

func createDataSourceName() string {
	opts := make(url.Values)
	opts.Add("_foreign_keys", "true")
	opts.Add("_journal_mode", viper.GetString("storage.database.journalmode"))

	dsn := url.URL{
		Scheme:   "file",
		Opaque:   viper.GetString("storage.database.filename"),
		RawQuery: opts.Encode(),
	}

	return dsn.String()
}

func loadChangesets() ([]groundwork.Changeset, error) {
	files, err := fs.Sub(changesetFolder, "changesets")
	if err != nil {
		return nil, fmt.Errorf("could not load changeset folder: %w", err)
	}

	return groundwork.FilesChangeset(files)
}
