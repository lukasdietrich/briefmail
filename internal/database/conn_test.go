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
	"testing"

	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDataSourceName(t *testing.T) {
	viper.Set("storage.database.filename", "somewhere/file.db")
	viper.Set("storage.database.journalmode", "off")

	dsn := createDataSourceName()
	assert.Equal(t, "file:somewhere/file.db?_foreign_keys=true&_journal_mode=off", dsn)
}

func TestOpenConnection(t *testing.T) {
	conn, err := openInMemory()
	require.NoError(t, err)
	require.NotNil(t, conn)

	rows, err := conn.QueryContext(context.Background(), "select 0 where 0 ;")
	require.NoError(t, err)
	require.NotNil(t, rows)

	assert.NoError(t, rows.Close())
	assert.NoError(t, conn.Close())
}

func openInMemory() (Conn, error) {
	viper.Set("storage.database.filename", ":memory:")
	viper.Set("storage.database.journalmode", "memory")

	return OpenConnection()
}

func TestBeginCommit(t *testing.T) {
	conn, err := openInMemory()
	require.NoError(t, err)
	require.NotNil(t, conn)

	defer conn.Close()

	var (
		ctx       = context.Background()
		domainDao = NewDomainDao()
	)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NotNil(t, tx)

	require.NoError(t, domainDao.Insert(ctx, tx, &models.DomainEntity{Name: "example.com"}))
	domains, err := domainDao.FindAll(ctx, tx)
	require.NoError(t, err)
	require.Len(t, domains, 1)

	require.NoError(t, tx.Commit())

	domains, err = domainDao.FindAll(ctx, conn)
	require.NoError(t, err)
	require.Len(t, domains, 1)
}

func TestBeginRollback(t *testing.T) {
	conn, err := openInMemory()
	require.NoError(t, err)
	require.NotNil(t, conn)

	defer conn.Close()

	var (
		ctx       = context.Background()
		domainDao = NewDomainDao()
	)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NotNil(t, tx)

	require.NoError(t, domainDao.Insert(ctx, tx, &models.DomainEntity{Name: "example.com"}))
	domains, err := domainDao.FindAll(ctx, tx)
	require.NoError(t, err)
	require.Len(t, domains, 1)

	require.NoError(t, tx.Rollback())

	domains, err = domainDao.FindAll(ctx, conn)
	require.NoError(t, err)
	require.Len(t, domains, 0)
}

func TestBeginRollbackWith(t *testing.T) {
	conn, err := openInMemory()
	require.NoError(t, err)
	require.NotNil(t, conn)

	defer conn.Close()

	var (
		ctx             = context.Background()
		domainDao       = NewDomainDao()
		callbackInvoked = false
	)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	require.NotNil(t, tx)

	require.NoError(t, domainDao.Insert(ctx, tx, &models.DomainEntity{Name: "example.com"}))
	domains, err := domainDao.FindAll(ctx, tx)
	require.NoError(t, err)
	require.Len(t, domains, 1)

	require.NoError(t, tx.RollbackWith(func() {
		callbackInvoked = true
	}))

	domains, err = domainDao.FindAll(ctx, conn)
	require.NoError(t, err)
	require.Len(t, domains, 0)

	assert.True(t, callbackInvoked)
}
