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

// InsertRecipient inserts a new recipient.
func InsertRecipient(tx *storage.Tx, recipient *storage.Recipient) error {
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

	result, err := tx.NamedExec(query, recipient)
	if err != nil {
		return err
	}

	recipient.ID, err = result.LastInsertId()
	return err
}

// UpdateRecipient updates an existing recipient.
func UpdateRecipient(tx *storage.Tx, recipient *storage.Recipient) error {
	const query = `
		update "recipients"
		set "mail_id"      = :mail_id ,
		    "mailbox_id"   = :mailbox_id ,
		    "forward_path" = :forward_path ,
		    "status"       = :status
		where "id" = :id ;
	`

	_, err := tx.NamedExec(query, recipient)
	return err
}

// UpdateRecipientsDelivered updates the status of all recipients matching the mail and mailbox to
// StatusDelivered.
func UpdateRecipientsDelivered(tx *storage.Tx, mailbox *storage.Mailbox, mail *storage.Mail) error {
	const query = `
		update "recipients"
		set "status" = $1
		where "mail_id" = $2
		  and "mailbox_id" = $3 ;
	`

	_, err := tx.Exec(query, storage.StatusDelivered, mail.ID, mailbox.ID)
	return err
}

// FindPendingRecipients returns all pending recipients of a mail.
func FindPendingRecipients(tx *storage.Tx, mail *storage.Mail) ([]storage.Recipient, error) {
	const query = `
		select *
		from "recipients"
		where "mail_id" = $1
		  and "status" = $2 ;
	`

	var recipientSlice []storage.Recipient
	return recipientSlice, tx.Select(&recipientSlice, query, mail.ID, storage.StatusPending)
}
