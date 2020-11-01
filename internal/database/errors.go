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
	"errors"

	"github.com/mattn/go-sqlite3"
)

// IsErrNoRows checks if an error is caused by an empty sql result set.
func IsErrNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// IsErrUnique checks if an error is caused by a unique constraint.
func IsErrUnique(err error) bool {
	return isErrSqliteExtended(err, sqlite3.ErrConstraintUnique) ||
		isErrSqliteExtended(err, sqlite3.ErrConstraintPrimaryKey)
}

func isErrSqliteExtended(err error, extendedCode sqlite3.ErrNoExtended) bool {
	var sqliteErr sqlite3.Error

	if errors.As(err, &sqliteErr) {
		return sqliteErr.ExtendedCode == extendedCode
	}

	return false
}
