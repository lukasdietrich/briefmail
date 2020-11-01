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

// RecipientDao is a data access object for all recipient related queries.
type RecipientDao interface {
	// Insert inserts a new recipient.
	Insert(context.Context, Queryer, *models.RecipientEntity) error
	// Update updates an existing recipient.
	Update(context.Context, Queryer, *models.RecipientEntity) error
	// UpdateDelivered updates the status of all recipients matching the mail and mailbox to
	// StatusDelivered.
	UpdateDelivered(context.Context, Queryer, *models.MailboxEntity, *models.MailEntity) error
	// FindPending returns all pending recipients of a mail.
	FindPending(context.Context, Queryer, *models.MailEntity) ([]models.RecipientEntity, error)
}

// recipientDao is the sqlite implementation of RecipientDao.
type recipientDao struct{}

// NewRecipientDao creates a new RecipientDao.
func NewRecipientDao() RecipientDao {
	return recipientDao{}
}

func (recipientDao) Insert(ctx context.Context, q Queryer, recipient *models.RecipientEntity) error {
	const query = `
		insert into "recipients" (
			"mail_id" ,
			"mailbox_id" ,
			"forward_path" ,
			"status"
		) values (
			:mail_id ,
			:mailbox_id ,
			:forward_path ,
			:status
		) ;
	`

	result, err := execNamed(ctx, q, query, recipient)
	if err != nil {
		return err
	}

	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	recipient.ID, err = result.LastInsertId()
	return err
}

func (recipientDao) Update(ctx context.Context, q Queryer, recipient *models.RecipientEntity) error {
	const query = `
		update "recipients"
		set "mail_id"      = :mail_id ,
		    "mailbox_id"   = :mailbox_id ,
		    "forward_path" = :forward_path ,
		    "status"       = :status
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, recipient)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (recipientDao) UpdateDelivered(
	ctx context.Context,
	q Queryer,
	mailbox *models.MailboxEntity,
	mail *models.MailEntity,
) error {
	const query = `
		update "recipients"
		set "status" = $1
		where "mail_id" = $2
		  and "mailbox_id" = $3 ;
	`

	result, err := execPositional(ctx, q, query, models.StatusDelivered, mail.ID, mailbox.ID)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (recipientDao) FindPending(
	ctx context.Context,
	q Queryer,
	mail *models.MailEntity,
) ([]models.RecipientEntity, error) {
	const query = `
		select *
		from "recipients"
		where "mail_id" = $1
		  and "status" = $2 ;
	`

	var recipientSlice []models.RecipientEntity

	if err := selectSlice(ctx, q, &recipientSlice, query, mail.ID, models.StatusPending); err != nil {
		return nil, err
	}

	return recipientSlice, nil
}
