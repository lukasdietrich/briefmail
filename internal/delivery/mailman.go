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

	"github.com/sirupsen/logrus"

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
	id, size, err := m.blobs.Write(content)
	if err != nil {
		return err
	}

	tx, err := m.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.RollbackWith(m.rollbackBlob(id))

	mail := storage.Mail{
		ID:         id,
		ReceivedAt: envelope.Date.Unix(),
		ReturnPath: envelope.From.String(),
		Size:       size,
	}

	logrus.Infof("delivering %q to %d recipients", id, len(envelope.To))

	if err := queries.InsertMail(tx, &mail); err != nil {
		return err
	}

	for _, to := range envelope.To {
		if err := m.deliverToRecipient(tx, to, &mail); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// rollbackBlob is used to rollback the mail blob, if an error occurs during delivery. Further
// errors happening inside of rollbackBlob are logged but not handled, because we do not want to
// shadow the original cause of the rollback.
func (m *Mailman) rollbackBlob(id string) func() {
	return func() {
		logrus.Info("an error occured during delivery, rolling back")

		if err := m.blobs.Delete(id); err != nil {
			logrus.Warnf("could not delete blob %q", id)
		}
	}
}

// deliverToRecipient determines if a recipient is local or not and acts
// accordingly. If local delivery fails because of a unique constraint, no
// error is returned. This can only occur when multiple addresses point to the
// same mailbox, in which case we just avoid duplicate entries.
func (m *Mailman) deliverToRecipient(tx *storage.Tx, to mails.Address, mail *storage.Mail) error {
	result, err := m.addressbook.LookupTx(tx, to)
	if err != nil {
		return err
	}

	switch {
	case result.IsLocal && result.Mailbox != nil:
		logrus.Debugf("adding %q to mailbox %d", mail.ID, result.Mailbox.ID)

		err := queries.InsertMailboxEntry(tx, result.Mailbox, mail)
		if !storage.IsErrUnique(err) {
			return err
		}

	case !result.IsLocal:
		// TODO: Add to outbound queue.
		logrus.Debugf("queueing %q for outbound delivery to %q", mail.ID, to)

	default:
		return fmt.Errorf("could not deliver to unknown address %q", to)
	}

	return nil
}
