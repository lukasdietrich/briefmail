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

	"github.com/jmoiron/sqlx"
)

type any = interface{}

func selectOne(ctx context.Context, q Queryer, dest any, query string, args ...any) error {
	return sqlx.GetContext(ctx, q, dest, query, args...)
}

func selectSlice(ctx context.Context, q Queryer, dest any, query string, args ...any) error {
	return sqlx.SelectContext(ctx, q, dest, query, args...)
}

func execPositional(ctx context.Context, q Queryer, query string, args ...any) (sql.Result, error) {
	return q.ExecContext(ctx, query, args...)
}

func execNamed(ctx context.Context, q Queryer, query string, arg any) (sql.Result, error) {
	return sqlx.NamedExecContext(ctx, q, query, arg)
}

func ensureRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
