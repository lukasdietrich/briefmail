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

// InsertMailbox inserts a new mailbox.
func InsertMailbox(tx *storage.Tx, mailbox *storage.Mailbox) error {
	const query = `
		insert into "mailboxes" (
			"display_name"
		) values (
			:display_name
		) ;
	`

	result, err := tx.NamedExec(query, mailbox)
	if err != nil {
		return err
	}

	mailbox.ID, err = result.LastInsertId()
	return err
}

// UpdateMailbox updates an existing mailbox.
func UpdateMailbox(tx *storage.Tx, mailbox *storage.Mailbox) error {
	const query = `
		update "mailboxes"
		set "display_name" = :display_name
		where "id" = :id ;
	`

	_, err := tx.NamedExec(query, mailbox)
	return err
}

// DeleteMailbox deletes an existing mailbox.
func DeleteMailbox(tx *storage.Tx, mailbox *storage.Mailbox) error {
	const query = `
		delete from "mailboxes"
		where "id" = :id ;
	`

	result, err := tx.NamedExec(query, mailbox)
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

// FindMailboxes returns all mailboxes.
func FindMailboxes(tx *storage.Tx) ([]storage.Mailbox, error) {
	const query = `
		select *
		from "mailboxes" ;
	`

	var mailboxSlice []storage.Mailbox
	if err := tx.Select(&mailboxSlice, query); err != nil {
		return nil, err
	}

	return mailboxSlice, nil
}

// FindMailboxByAddress returns the mailbox of the address matching the local-part and domain.
func FindMailboxByAddress(tx *storage.Tx, localPart, domain string) (*storage.Mailbox, error) {
	const query = `
		select "mailboxes".*
		from "mailboxes"
			inner join "addresses" on "mailboxes"."id" = "addresses"."mailbox_id"
			inner join "domains" on "domains"."id" = "addresses"."domain_id"
		where "addresses"."local_part" = $1
		  and "domains"."name" = $2
		limit 1 ;
	`

	var mailbox storage.Mailbox

	if err := tx.Get(&mailbox, query, localPart, domain); err != nil {
		return nil, err
	}

	return &mailbox, nil
}
