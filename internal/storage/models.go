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

import "database/sql"

// Mailbox is the entity for the "mailboxes" table.
type Mailbox struct {
	ID   int64  `db:"id"`
	Hash string `db:"hash"`
}

// Mail is the entity for the "mails" table.
type Mail struct {
	ID         string        `db:"id"`
	ReceivedAt int64         `db:"received_at"`
	DeletedAt  sql.NullInt64 `db:"deleted_at"`
	ReturnPath string        `db:"return_path"`
	Size       int64         `db:"size"`
	Attempt    int           `db:"attempt"`
}

// DeliveryStatus indicates the status of delivery per recipient.
type DeliveryStatus int

// The gaps between the numerical values is in case we later need to add some new ones in between.

const (
	// StatusFailed is a mail that could not be delivered after the final attempt.
	StatusFailed = DeliveryStatus(10)
	// StatusDelivered is a mail that reached its final destination. This is either a successful
	// outbound transmission or a local mail, that has been retrieved.
	StatusDelivered = DeliveryStatus(20)
	// StatusInboxed is a mail delivered, but not deleted, to a local mailbox.
	StatusInboxed = DeliveryStatus(30)
	// StatusPending is a mail queued for outbound transmision.
	StatusPending = DeliveryStatus(40)
	// MaxCompletedStatus is the highest (numerical) status, that is considered "completed".
	MaxCompletedStatus = StatusDelivered
)

// Recipient is the entity for the "recipients" table.
type Recipient struct {
	ID          int64          `db:"id"`
	MailID      string         `db:"mail_id"`
	MailboxID   sql.NullInt64  `db:"mailbox_id"`
	ForwardPath string         `db:"forward_path"`
	Status      DeliveryStatus `db:"status"`
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
