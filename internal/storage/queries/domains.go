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

package queries

import (
	"database/sql"

	"github.com/lukasdietrich/briefmail/internal/storage"
)

// InsertDomain inserts a new domain.
func InsertDomain(tx *storage.Tx, domain *storage.Domain) error {
	const query = `
		insert into "domains" ( "name" )
		values ( :name ) ;
	`

	result, err := tx.NamedExec(query, domain)
	if err != nil {
		return err
	}

	domain.ID, err = result.LastInsertId()
	return err
}

// DeleteDomain removes an existing domain.
func DeleteDomain(tx *storage.Tx, name string) error {
	const query = `
		delete from "domains"
		where "name" = $1 ;
	`

	result, err := tx.Exec(query, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ListDomains returns all domains, sorted by name.
func ListDomains(tx *storage.Tx) ([]storage.Domain, error) {
	const query = `
		select *
		from "domains"
		order by "name" asc ;
	`

	var domainSlice []storage.Domain
	return domainSlice, tx.Select(&domainSlice, query)
}

// ExistsDomain checks if a domain is already inserted.
func ExistsDomain(tx *storage.Tx, name string) (bool, error) {
	const query = `
		select exists (
			select 1
			from "domains"
			where "name" = $1 
			limit 1
		) ;
	`

	var exists bool

	if err := tx.Get(&exists, query, name); err != nil {
		return false, err
	}

	return exists, nil
}

func FindDomain(tx *storage.Tx, name string) (*storage.Domain, error) {
	const query = `
		select *
		from "domains"
		where "name" = $1
		limit 1 ;
	`

	var domain storage.Domain
	return &domain, tx.Get(&domain, query, name)
}
