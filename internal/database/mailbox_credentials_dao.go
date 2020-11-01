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

// MailboxCredentialDao is a data access object for all mailbox-credential related queries.
type MailboxCredentialDao interface {
	// Upsert inserts new mailbox credentials. When there already is an entry for a mailbox, the
	// row will be updated instead.
	Upsert(context.Context, Queryer, *models.MailboxCredentialEntity) error
	// FindByMailbox returns the credentials associated with a mailbox.
	FindByMailbox(context.Context, Queryer, *models.MailboxEntity) (*models.MailboxCredentialEntity, error)
}

// mailboxCredentialDao is the sqlite implementation of MailboxCredentialDao.
type mailboxCredentialDao struct{}

// NewMailboxCredentialDao creates a new MailboxCredentialDao.
func NewMailboxCredentialDao() MailboxCredentialDao {
	return mailboxCredentialDao{}
}

func (mailboxCredentialDao) Upsert(
	ctx context.Context,
	q Queryer,
	creds *models.MailboxCredentialEntity,
) error {
	const query = `
		insert or replace into "mailbox_credentials" (
			"mailbox_id" ,
			"updated_at" ,
			"hash"
		) values (
			:mailbox_id ,
			:updated_at ,
			:hash
		) ;
	`

	result, err := execNamed(ctx, q, query, creds)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (mailboxCredentialDao) FindByMailbox(
	ctx context.Context,
	q Queryer,
	mailbox *models.MailboxEntity,
) (*models.MailboxCredentialEntity, error) {
	const query = `
		select *
		from "mailbox_credentials"
		where "mailbox_id" = $1 ;
	`

	var creds models.MailboxCredentialEntity

	if err := selectOne(ctx, q, &creds, query, mailbox.ID); err != nil {
		return nil, err
	}

	return &creds, nil
}
