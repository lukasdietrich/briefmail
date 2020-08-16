package storage

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
	return isErrSqliteExtended(err, sqlite3.ErrConstraintUnique)
}

//
func isErrSqliteExtended(err error, extendedCode sqlite3.ErrNoExtended) bool {
	var sqliteErr sqlite3.Error

	return errors.As(err, &sqliteErr) &&
		sqliteErr.ExtendedCode == extendedCode
}
