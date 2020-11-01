// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
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

package delivery

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

// Envelope stores the information about an email before the actual content is
// read. It is basically what a real envelope is to mail.
type Envelope struct {
	// Helo is the string provided by an smtp client when greeting the server.
	Helo string
	// Addr is the remote address of the sender.
	Addr net.IP
	// Date is the time when the data transmission begins.
	Date time.Time
	// From is the email-address of the sender.
	From models.Address
	// To is a list of recipient email-addresses.
	To []models.Address
}

// Mailman handles the delivery of local mails into mailboxes as well as queuing outbound delivery.
type Mailman struct {
	database     database.Conn
	mailDao      database.MailDao
	recipientDao database.RecipientDao
	blobs        *storage.Blobs
	addressbook  *Addressbook
	queue        *Queue
}

// NewMailman creates a new mailman for delivery.
func NewMailman(
	db database.Conn,
	mailDao database.MailDao,
	recipientDao database.RecipientDao,
	blobs *storage.Blobs,
	addressbook *Addressbook,
	queue *Queue,
) *Mailman {
	return &Mailman{
		database:     db,
		mailDao:      mailDao,
		recipientDao: recipientDao,
		blobs:        blobs,
		addressbook:  addressbook,
		queue:        queue,
	}
}

// Deliver goes through the list of recipients and determines if they are local
// or outbound. Local mails are put into the corresponding mailboxes. Outbound
// mails are queued for delivery. Because of the queue errors during outbound
// delivery are not known at this point.
func (m *Mailman) Deliver(ctx context.Context, envelope Envelope, content io.Reader) error {
	id, size, err := m.blobs.Write(ctx, content)
	if err != nil {
		return err
	}

	tx, err := m.database.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.RollbackWith(m.rollbackBlob(ctx, id))

	mail := models.MailEntity{
		ID:         id,
		ReceivedAt: envelope.Date.Unix(),
		ReturnPath: envelope.From,
		Size:       size,
	}

	log.InfoContext(ctx).
		Stringer("from", envelope.From).
		Int("recipients", len(envelope.To)).
		Str("mail", id).
		Msg("delivering mail to recipients")

	if err := m.mailDao.Insert(ctx, tx, &mail); err != nil {
		return err
	}

	for _, to := range envelope.To {
		if err := m.deliverTo(ctx, tx, &mail, to); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	m.queue.WakeUp(ctx)
	return nil
}

// rollbackBlob is used to rollback the mail blob, if an error occurs during delivery. Further
// errors happening inside of rollbackBlob are logged but not handled, because we do not want to
// shadow the original cause of the rollback.
func (m *Mailman) rollbackBlob(ctx context.Context, id string) func() {
	return func() {
		log.ErrorContext(ctx).
			Str("mail", id).
			Msg("an error occured during delivery, rolling back")

		if err := m.blobs.Delete(ctx, id); err != nil {
			log.ErrorContext(ctx).
				Str("mail", id).
				Err(err).
				Msg("could not delete mail blob")
		}
	}
}

// deliverTo delivers a mail to a single recipient. It determines if a recipient is local or not and
// acts accordingly. Local mails are immediatey put into the associated mailbox. Outbound mails are
// queued for later transmission.
func (m *Mailman) deliverTo(
	ctx context.Context,
	tx database.Tx,
	mail *models.MailEntity,
	to models.Address,
) error {
	result, err := m.addressbook.LookupTx(ctx, tx, to)
	if err != nil {
		return err
	}

	recipient := models.RecipientEntity{
		MailID:      mail.ID,
		ForwardPath: to,
	}

	switch {
	case result.IsLocal && result.Mailbox != nil:
		log.InfoContext(ctx).
			Str("mail", mail.ID).
			Int64("mailbox", result.Mailbox.ID).
			Stringer("to", to).
			Msg("delivering mail to local mailbox")

		recipient.MailboxID.Int64 = result.Mailbox.ID
		recipient.MailboxID.Valid = true
		recipient.Status = models.StatusInboxed

	case !result.IsLocal:
		log.InfoContext(ctx).
			Str("mail", mail.ID).
			Stringer("to", to).
			Msg("queueing mail for outbound delivery")

		recipient.Status = models.StatusPending

	default:
		return fmt.Errorf("could not deliver to unknown address %q", to)
	}

	return m.recipientDao.Insert(ctx, tx, &recipient)
}
