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

package models

import (
	"database/sql"
)

// MailboxEntity is the entity for the "mailboxes" table.
type MailboxEntity struct {
	ID          int64  `db:"id"`
	DisplayName string `db:"display_name"`
}

// MailboxCredentialEntity is the entity for the "mailbox_credentials" table.
type MailboxCredentialEntity struct {
	MailboxID int64  `db:"mailbox_id"`
	UpdatedAt int64  `db:"updated_at"`
	Hash      string `db:"hash"`
}

// MailEntity is the entity for the "mails" table.
type MailEntity struct {
	ID              string        `db:"id"`
	ReceivedAt      int64         `db:"received_at"`
	DeletedAt       sql.NullInt64 `db:"deleted_at"`
	ReturnPath      Address       `db:"return_path"`
	Size            int64         `db:"size"`
	AttemptCount    int           `db:"attempt_count"`
	LastAttemptedAt sql.NullInt64 `db:"last_attempted_at"`
}

// DeliveryStatus indicates the status of delivery per recipient.
type DeliveryStatus int

const (
	_ DeliveryStatus = iota
	// StatusFailed is a mail that could not be delivered after the final attempt.
	StatusFailed
	// StatusDelivered is a mail that reached its final destination. This is either a successful
	// outbound transmission or a local mail, that has been retrieved.
	StatusDelivered
	// StatusInboxed is a mail delivered, but not deleted, to a local mailbox.
	StatusInboxed
	// StatusPending is a mail queued for outbound transmision.
	StatusPending
)

// RecipientEntity is the entity for the "recipients" table.
type RecipientEntity struct {
	ID          int64          `db:"id"`
	MailID      string         `db:"mail_id"`
	MailboxID   sql.NullInt64  `db:"mailbox_id"`
	ForwardPath Address        `db:"forward_path"`
	Status      DeliveryStatus `db:"status"`
}

// DomainEntity is the entity for the "domains" table.
type DomainEntity struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// AddressEntity is the entity for the "addresses" table.
type AddressEntity struct {
	ID        int64  `db:"id"`
	LocalPart string `db:"local_part"`
	DomainID  int64  `db:"domain_id"`
	MailboxID int64  `db:"mailbox_id"`
}
