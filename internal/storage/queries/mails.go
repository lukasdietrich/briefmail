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
		insert into "mails" ( "id", "received_at", "return_path", "size" )
		values ( :id, :received_at, :return_path, :size ) ;
	`

	_, err := tx.NamedExec(query, mail)
	return err
}

// DeleteMail removes an existing mail.
func DeleteMail(tx *storage.Tx, mail *storage.Mail) error {
	const query = `
		delete from "mails"
		where "id" = :id
		limit 1 ;
	`

	_, err := tx.NamedExec(query, mail)
	return err
}

// InsertMailboxEntry inserts a mail into a mailbox.
func InsertMailboxEntry(tx *storage.Tx, mailbox *storage.Mailbox, mail *storage.Mail) error {
	const query = `
		insert into "mailbox_entries" ( "mailbox_id", "mail_id"  )
		values ( $1, $2 ) ;
	`

	_, err := tx.Exec(query, mailbox.ID, mail.ID)
	return err
}

// DeleteMailboxEntry deletes a mail from an inbox. This does not remove the mail itself.
func DeleteMailboxEntry(tx *storage.Tx, mailbox *storage.Mailbox, mail *storage.Mail) error {
	const query = `
		delete from "mailbox_entries"
		where "mailbox_id" = $1
		  and "mail_id" = $2 ;
	`

	_, err := tx.Exec(query, mailbox.ID, mail.ID)
	return err
}

// FindMailsByMailbox returns a slice of mails in the mailbox sorted by date.
func FindMailsByMailbox(tx *storage.Tx, mailbox *storage.Mailbox) ([]storage.Mail, error) {
	const query = `
		select "mails".*
		from "mails"
			inner join "mailbox_entries"
				on "mails"."id" = "mailbox_entries"."mail_id"
		where "mailbox_entries"."mailbox_id" = $1
		order by "mails"."received_at" asc ;
	`

	var mailSlice []storage.Mail
	return mailSlice, tx.Select(&mailSlice, query, mailbox.ID)
}

// FindOrphanedMails returns a slice of mails that are not in any mailbox.
func FindOrphanedMails(tx *storage.Tx) ([]storage.Mail, error) {
	const query = `
		select "mails".*
		from "mails"
			left join "mailbox_entries"
				on "mails"."id" = "mailbox_entries"."mail_id"
		where "mailbox_entries"."mail_id" is null ;
	`

	var mailSlice []storage.Mail
	return mailSlice, tx.Select(&mailSlice, query)
}
