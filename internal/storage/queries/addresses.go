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

import "github.com/lukasdietrich/briefmail/internal/storage"

// InsertAddress inserts a new address.
func InsertAddress(tx *storage.Tx, address *storage.Address) error {
	const query = `
		insert into "addresses" ( "local_part", "domain_id", "mailbox_id" )
		values ( :local_part, :domain_id, :mailbox_id ) ;
	`

	result, err := tx.NamedExec(query, address)
	if err != nil {
		return err
	}

	address.ID, err = result.LastInsertId()
	return err
}

// ExistsAddress checks if an address exists.
func ExistsAddress(tx *storage.Tx, localPart, domain string) (bool, error) {
	const query = `
		select exists (
			select 1
			from "addresses" inner join "domains" on "addresses"."domain_id" = "domains"."id"
			where "addresses"."local_part" = $1
			  and "domains"."name" = $2 
			limit 1 ;
		) ;
	`

	var exists bool

	if err := tx.Get(&exists, query, localPart, domain); err != nil {
		return false, err
	}

	return exists, nil
}

// FindAddress returns the address matching both local-part and domain.
func FindAddress(tx *storage.Tx, localPart, domain string) (*storage.Address, error) {
	const query = `
		select "addresses".*
		from "addresses" inner join "domains" on "addresses"."domain_id" = "domains"."id"
		where "addresses"."local_part" = $1
		  and "domains"."name" = $2
		limit 1 ;
	`

	var address storage.Address
	return &address, tx.Get(&address, query, localPart, domain)
}

func ListAddressesByDomain(tx *storage.Tx, domain string) ([]storage.Address, error) {
	const query = `
		select "addresses".*
		from "addresses" inner join "domains" on "addresses"."domain_id" = "domains"."id"
		where "domains"."name" = $1 ;
	`

	var addressSlice []storage.Address
	return addressSlice, tx.Select(&addressSlice, query, domain)
}
