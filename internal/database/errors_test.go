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
	"database/sql"
	"fmt"
	"testing"

	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestIsErrNoRows(t *testing.T) {
	for err, ok := range map[error]bool{
		sql.ErrNoRows:                        true,
		fmt.Errorf("foo: %w", sql.ErrNoRows): true,
		sql.ErrConnDone:                      false,
	} {
		assert.Equal(t, ok, IsErrNoRows(err))
	}
}

func TestIsErrUnique(t *testing.T) {
	for err, ok := range map[error]bool{
		sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintUnique}:     true,
		sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintPrimaryKey}: true,
		sql.ErrNoRows: false,
	} {
		assert.Equal(t, ok, IsErrUnique(err))
	}
}
