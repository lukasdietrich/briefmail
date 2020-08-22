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

package storage

import (
	"context"
	"database/sql"
	"errors"
	"net/url"

	rice "github.com/GeertJohan/go.rice"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
)

const driverName = "sqlite3"

// Database is a connection to the sql database.
type Database struct {
	conn *sqlx.DB
}

func init() {
	migrate.SetTable("migrations")

	viper.SetDefault("storage.database.filename", "data/briefmail.sqlite")
	viper.SetDefault("storage.database.journalmode", "wal")
}

// OpenDatabase opens a sqlite3 database using the configuration from viper.
//
// `storage.database.filename` is the filename for the sqlite database.
// `storage.database.journalmode` will be used for the journalmode pragma.
func OpenDatabase() (*Database, error) {
	dsn := createDataSourceName()
	log.Info().
		Str("dataSourceName", dsn).
		Msg("connecting to database")

	db, err := sqlx.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	n, err := migrate.Exec(db.DB, driverName, migrations, migrate.Up)
	if err != nil {
		return nil, err
	}

	if n > 0 {
		log.Info().
			Int("migrations", n).
			Msg("database migrations applied")
	}

	return &Database{db}, nil
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

func loadMigrations() (migrate.MigrationSource, error) {
	box, err := rice.FindBox("../../migrations")
	if err != nil {
		return nil, err
	}

	source := migrate.HttpFileSystemMigrationSource{
		FileSystem: box.HTTPBox(),
	}

	return &source, nil
}

// BeginTx starts a new database transaction. Every call to Exec, Query, Get or Select uses passes
// the context provided to this method.
func (d *Database) BeginTx(ctx context.Context) (*Tx, error) {
	raw, err := d.conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &Tx{raw, ctx}, nil
}

// Tx is a database transaction. Unlike the standard *sql.Tx, this one also wraps the context used
// in to begin the transaction.
type Tx struct {
	raw *sqlx.Tx
	ctx context.Context
}

// Rollback rolls back the transaction.
func (t *Tx) Rollback() error {
	return t.raw.Rollback()
}

// RollbackWith calls Rollback and, unless the transaction was already committed, calls the
// callback function.
func (t *Tx) RollbackWith(callback func()) error {
	err := t.Rollback()

	if !errors.Is(err, sql.ErrTxDone) {
		callback()
	}

	return err
}

// Commit commits the transaction.
func (t *Tx) Commit() error {
	return t.raw.Commit()
}

// Exec executes a query that does not return rows.
func (t *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.raw.ExecContext(t.ctx, query, args...)
}

// NamedExec executes a query and maps the query parameters by name.
func (t *Tx) NamedExec(query string, args interface{}) (sql.Result, error) {
	return t.raw.NamedExecContext(t.ctx, query, args)
}

// Get executes a query returning a single row.
func (t *Tx) Get(dest interface{}, query string, args ...interface{}) error {
	return t.raw.GetContext(t.ctx, dest, query, args...)
}

// Select executes a query returning multiple rows.
func (t *Tx) Select(dest interface{}, query string, args ...interface{}) error {
	return t.raw.SelectContext(t.ctx, dest, query, args...)
}
