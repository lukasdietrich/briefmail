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

// MailDao is a data access object for all mail related queries.
type MailDao interface {
	// Insert inserts a new mail.
	Insert(context.Context, Queryer, *models.MailEntity) error
	// Update updates an existing mail.
	Update(context.Context, Queryer, *models.MailEntity) error
	// FindByMailbox returns all mails that are not deleted and are "inboxed" to the mailbox.
	FindByMailbox(context.Context, Queryer, *models.MailboxEntity) ([]models.MailEntity, error)
	// FindDeletable returns all mails which are not yet deleted and are delivered or failed to all
	// recipients.
	FindDeletable(context.Context, Queryer) ([]models.MailEntity, error)
	// FindNextPending returns the next mail with at least one pending recipient.
	FindNextPending(context.Context, Queryer) (*models.MailEntity, error)
}

// mailDao is the sqlite implementation of MailDao.
type mailDao struct{}

// NewMailDao creates a new MailDao.
func NewMailDao() MailDao {
	return mailDao{}
}

func (mailDao) Insert(ctx context.Context, q Queryer, mail *models.MailEntity) error {
	const query = `
		insert into "mails" (
			"id" ,
			"received_at" ,
			"deleted_at" ,
			"return_path" ,
			"size" ,
			"attempt_count" ,
			"last_attempted_at"
		) values (
			:id ,
			:received_at ,
			:deleted_at ,
			:return_path ,
			:size ,
			:attempt_count ,
			:last_attempted_at
		) ;
	`

	result, err := execNamed(ctx, q, query, mail)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (mailDao) Update(ctx context.Context, q Queryer, mail *models.MailEntity) error {
	const query = `
		update "mails"
		set "received_at"       = :received_at ,
			"deleted_at"        = :deleted_at ,
			"return_path"       = :return_path ,
			"size"              = :size ,
			"attempt_count"     = :attempt_count ,
			"last_attempted_at" = :last_attempted_at
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, mail)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (mailDao) FindByMailbox(
	ctx context.Context,
	q Queryer,
	mailbox *models.MailboxEntity,
) ([]models.MailEntity, error) {
	const query = `
		select distinct "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		  and "recipients"."mailbox_id" = $1
		  and "recipients"."status" = $2
		order by "mails"."received_at" asc ;
	`

	var mailSlice []models.MailEntity

	if err := selectSlice(ctx, q, &mailSlice, query, mailbox.ID, models.StatusInboxed); err != nil {
		return nil, err
	}

	return mailSlice, nil
}

func (mailDao) FindDeletable(ctx context.Context, q Queryer) ([]models.MailEntity, error) {
	const query = `
		select "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		group by "mails"."id"
		having count(iif("recipients"."status" in ($1, $2), null, 1)) = 0
		order by "mails"."received_at" asc ;
	`

	var mailSlice []models.MailEntity

	err := selectSlice(ctx, q, &mailSlice, query,
		models.StatusDelivered,
		models.StatusFailed)

	if err != nil {
		return nil, err
	}

	return mailSlice, nil
}

func (mailDao) FindNextPending(ctx context.Context, q Queryer) (*models.MailEntity, error) {
	const query = `
		select "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		  and "recipients"."status" = $1
		order by "mails"."last_attempted_at" asc ,
		         "mails"."attempt_count" asc ,
		         "mails"."received_at" asc
		limit 1 ;
	`

	var mail models.MailEntity

	if err := selectOne(ctx, q, &mail, query, models.StatusPending); err != nil {
		return nil, err
	}

	return &mail, nil
}
