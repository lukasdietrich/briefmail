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

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

// Mailman is handles the delivery of local mails into mailboxes as well as
// queuing outbound delivery.
type Mailman struct {
	database    *storage.Database
	blobs       *storage.Blobs
	addressbook *Addressbook
}

// NewMailman creates a new mailman for delivery.
func NewMailman(
	database *storage.Database,
	blobs *storage.Blobs,
	addressbook *Addressbook,
) *Mailman {
	return &Mailman{
		database:    database,
		blobs:       blobs,
		addressbook: addressbook,
	}
}

// Deliver goes through the list of recipients and determines if they are local
// or outbound. Local mails are put into the corresponding mailboxes. Outbound
// mails are queued for delivery. Because of the queue errors during outbound
// delivery are not known at this point.
func (m *Mailman) Deliver(ctx context.Context, envelope mails.Envelope, content io.Reader) error {
	id, size, err := m.blobs.Write(ctx, content)
	if err != nil {
		return err
	}

	tx, err := m.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.RollbackWith(m.rollbackBlob(ctx, id))

	mail := storage.Mail{
		ID:         id,
		ReceivedAt: envelope.Date.Unix(),
		ReturnPath: envelope.From.String(),
		Size:       size,
	}

	log.InfoContext(ctx).
		Stringer("from", envelope.From).
		Int("recipients", len(envelope.To)).
		Str("mail", id).
		Msg("delivering mail to recipients")

	if err := queries.InsertMail(tx, &mail); err != nil {
		return err
	}

	for _, to := range envelope.To {
		if err := m.deliverTo(ctx, tx, mail.ID, to); err != nil {
			return err
		}
	}

	return tx.Commit()
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
func (m *Mailman) deliverTo(ctx context.Context, tx *storage.Tx, mailID string, to mails.Address) error {
	result, err := m.addressbook.LookupTx(tx, to)
	if err != nil {
		return err
	}

	recipient := storage.Recipient{
		MailID:      mailID,
		ForwardPath: to.String(),
	}

	switch {
	case result.IsLocal && result.Mailbox != nil:
		log.InfoContext(ctx).
			Str("mail", mailID).
			Int64("mailbox", result.Mailbox.ID).
			Stringer("to", to).
			Msg("delivering mail to local mailbox")

		recipient.MailboxID.Int64 = result.Mailbox.ID
		recipient.MailboxID.Valid = true
		recipient.Status = storage.StatusInboxed

	case !result.IsLocal:
		log.InfoContext(ctx).
			Str("mail", mailID).
			Stringer("to", to).
			Msg("queueing mail for outbound delivery")

		recipient.Status = storage.StatusPending

	default:
		return fmt.Errorf("could not deliver to unknown address %q", to)
	}

	return queries.InsertRecipient(tx, &recipient)
}
