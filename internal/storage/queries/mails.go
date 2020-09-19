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
	"github.com/lukasdietrich/briefmail/internal/storage"
)

// InsertMail inserts a new mail.
func InsertMail(tx *storage.Tx, mail *storage.Mail) error {
	const query = `
		insert into "mails" (
			"id" ,
			"received_at" ,
			"deleted_at" ,
			"return_path" ,
			"size" ,
			"attempts" ,
			"last_attempted_at"
		) values (
			:id ,
			:received_at ,
			:deleted_at ,
			:return_path ,
			:size ,
			:attempts ,
			:last_attempted_at
		) ;
	`

	_, err := tx.NamedExec(query, mail)
	return err
}

// UpdateMail updates an existing mail.
func UpdateMail(tx *storage.Tx, mail *storage.Mail) error {
	const query = `
		update "mails"
		set "received_at"       = :received_at ,
			"deleted_at"        = :deleted_at ,
			"return_path"       = :return_path ,
			"size"              = :size ,
			"attempts"          = :attempts ,
			"last_attempted_at" = :last_attempted_at
		where "id" = :id ;
	`

	_, err := tx.NamedExec(query, mail)
	return err
}

// FindMailsByMailbox returns all mails that are not deleted and are "inboxed" to the mailbox.
func FindMailsByMailbox(tx *storage.Tx, mailbox *storage.Mailbox) ([]storage.Mail, error) {
	const query = `
		select distinct "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		  and "recipients"."mailbox_id" = $1
		  and "recipients"."status" = $2
		order by "mails"."received_at" asc ;
	`

	var mailSlice []storage.Mail
	return mailSlice, tx.Select(&mailSlice, query, mailbox.ID, storage.StatusInboxed)
}

// FindDeletableMails returns all mails which are not yet deleted and are delivered or failed to all
// recipients.
func FindDeletableMails(tx *storage.Tx) ([]storage.Mail, error) {
	const query = `
		select "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		group by "mails"."id"
		having count(iif("recipients"."status" in ($1, $2), null, 1)) = 0
		order by "mails"."received_at" asc ;
	`

	var mailSlice []storage.Mail
	return mailSlice, tx.Select(&mailSlice, query, storage.StatusDelivered, storage.StatusFailed)
}

// FindNextPendingMail returns the next mail with at least one pending recipient.
func FindNextPendingMail(tx *storage.Tx) (*storage.Mail, error) {
	const query = `
		select "mails".*
		from "mails" inner join "recipients" on "mails"."id" = "recipients"."mail_id"
		where "mails"."deleted_at" is null
		  and "recipients"."status" = $1
		order by "mails"."last_attempted_at" asc ,
		         "mails"."attempts" asc ,
		         "mails"."received_at" asc
		limit 1 ;
	`

	var mail storage.Mail
	if err := tx.Get(&mail, query, storage.StatusPending); err != nil {
		return nil, err
	}

	return &mail, nil
}
