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

// MailboxDao is a data access object for all mailbox related queries.
type MailboxDao interface {
	// Insert inserts a new mailbox.
	Insert(context.Context, Queryer, *models.MailboxEntity) error
	// Update updates an existing mailbox.
	Update(context.Context, Queryer, *models.MailboxEntity) error
	// DeleteMailbox deletes an existing mailbox.
	Delete(context.Context, Queryer, *models.MailboxEntity) error
	// FindAll returns all mailboxes.
	FindAll(context.Context, Queryer) ([]models.MailboxEntity, error)
	// FindByAddress returns the mailbox associated with an address.
	FindByAddress(context.Context, Queryer, models.Address) (*models.MailboxEntity, error)
}

// mailboxDao is the sqlite implementation of MailboxDao.
type mailboxDao struct{}

// NewMailboxDao creates a new MailboxDao.
func NewMailboxDao() MailboxDao {
	return mailboxDao{}
}

func (mailboxDao) Insert(ctx context.Context, q Queryer, mailbox *models.MailboxEntity) error {
	const query = `
		insert into "mailboxes" (
			"display_name"
		) values (
			:display_name
		) ;
	`

	result, err := execNamed(ctx, q, query, mailbox)
	if err != nil {
		return err
	}

	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	mailbox.ID, err = result.LastInsertId()
	return err
}

func (mailboxDao) Update(ctx context.Context, q Queryer, mailbox *models.MailboxEntity) error {
	const query = `
		update "mailboxes"
		set "display_name" = :display_name
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, mailbox)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (mailboxDao) Delete(ctx context.Context, q Queryer, mailbox *models.MailboxEntity) error {
	const query = `
		delete from "mailboxes"
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, mailbox)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (mailboxDao) FindAll(ctx context.Context, q Queryer) ([]models.MailboxEntity, error) {
	const query = `
		select *
		from "mailboxes" ;
	`

	var mailboxSlice []models.MailboxEntity

	if err := selectSlice(ctx, q, &mailboxSlice, query); err != nil {
		return nil, err
	}

	return mailboxSlice, nil
}

func (mailboxDao) FindByAddress(
	ctx context.Context,
	q Queryer,
	address models.Address,
) (*models.MailboxEntity, error) {
	const query = `
		select "mailboxes".*
		from "mailboxes"
			inner join "addresses" on "mailboxes"."id" = "addresses"."mailbox_id"
			inner join "domains" on "domains"."id" = "addresses"."domain_id"
		where "addresses"."local_part" = $1
		  and "domains"."name" = $2
		limit 1 ;
	`

	var (
		mailbox    models.MailboxEntity
		localPart  = address.LocalPart()
		domainName = address.Domain()
	)

	if err := selectOne(ctx, q, &mailbox, query, localPart, domainName); err != nil {
		return nil, err
	}

	return &mailbox, nil
}
