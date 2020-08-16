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

package storage

// Mailbox is the entity for the "mailboxes" table.
type Mailbox struct {
	ID   int64  `db:"id"`
	Hash string `db:"hash"`
}

// Mail is the entity for the "mails" table.
type Mail struct {
	ID         string `db:"id"`
	ReceivedAt int64  `db:"received_at"`
	ReturnPath string `db:"return_path"`
	Size       int64  `db:"size"`
}

// Domain is the entity for the "domains" table.
type Domain struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// Address is the entity for the "addresses" table.
type Address struct {
	ID        int64  `db:"id"`
	LocalPart string `db:"local_part"`
	DomainID  int64  `db:"domain_id"`
	MailboxID int64  `db:"mailbox_id"`
}
