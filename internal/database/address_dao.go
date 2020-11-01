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

	"github.com/lukasdietrich/briefmail/internal/models"
)

// AddressWithDomain is a helper type to eagerly fetch the domain name of an address.
type AddressWithDomain struct {
	models.AddressEntity
	DomainName string `db:"domain_name"`
}

// AddressDao is a data access object for all address related queries.
type AddressDao interface {
	// Insert inserts a new address.
	Insert(context.Context, Queryer, *models.AddressEntity) error
	// Delete deletes an existing address.
	Delete(context.Context, Queryer, *models.AddressEntity) error
	// FindAll returns all addresses including their domain name.
	FindAll(context.Context, Queryer) ([]AddressWithDomain, error)
	// FindByMailbox returns all addresses including their domain name by mailbox.
	FindByMailbox(context.Context, Queryer, *models.MailboxEntity) ([]AddressWithDomain, error)
}

// addressDao is the sqlite implementation of AddressDao.
type addressDao struct{}

// NewAddressDao creates a new AddressDao.
func NewAddressDao() AddressDao {
	return addressDao{}
}

func (addressDao) Insert(ctx context.Context, q Queryer, address *models.AddressEntity) error {
	const query = `
		insert into "addresses" (
			"local_part" ,
			"domain_id" ,
			"mailbox_id"
		) values (
			:local_part ,
			:domain_id ,
			:mailbox_id
		) ;
	`

	result, err := execNamed(ctx, q, query, address)
	if err != nil {
		return err
	}

	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	address.ID, err = result.LastInsertId()
	return err
}

func (addressDao) Delete(ctx context.Context, q Queryer, address *models.AddressEntity) error {
	const query = `
		delete from "addresses"
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, address)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (addressDao) FindAll(ctx context.Context, q Queryer) ([]AddressWithDomain, error) {
	const query = `
		select "addresses".*, "domains"."name" as "domain_name"
		from "addresses" inner join "domains" on "addresses"."domain_id" = "domains"."id" ;
	`

	var addressSlice []AddressWithDomain

	if err := selectSlice(ctx, q, &addressSlice, query); err != nil {
		return nil, err
	}

	return addressSlice, nil
}

func (addressDao) FindByMailbox(
	ctx context.Context,
	q Queryer,
	mailbox *models.MailboxEntity,
) ([]AddressWithDomain, error) {
	const query = `
		select "addresses".*, "domains"."name" as "domain_name"
		from "addresses" inner join "domains" on "addresses"."domain_id" = "domains"."id"
		where "addresses"."mailbox_id" = $1 ;
	`

	var addressSlice []AddressWithDomain

	if err := selectSlice(ctx, q, &addressSlice, query, mailbox.ID); err != nil {
		return nil, err
	}

	return addressSlice, nil
}
