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
	"fmt"
	"io"

	"github.com/lukasdietrich/briefmail/internal/addressbook"
	"github.com/lukasdietrich/briefmail/internal/model"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

type Mailman struct {
	DB          *storage.DB
	Blobs       *storage.Blobs
	Addressbook addressbook.Addressbook
	Queue       *QueueWorker
}

func (m *Mailman) Deliver(envelope *model.Envelope, mail io.Reader) error {
	id, size, err := m.Blobs.Write(mail)
	if err != nil {
		return err
	}

	if err := m.DB.AddMail(id, size, 0, envelope); err != nil {
		m.Blobs.Delete(id)
		return err
	}

	var (
		mailboxes []int64
		queue     []*model.Address
	)

	for _, addr := range envelope.To {
		entry := m.Addressbook.Lookup(addr)
		if entry == nil {
			return fmt.Errorf("could not deliver to %s", addr)
		}

		switch entry.Kind {
		case addressbook.Local:
			mailboxes = append(mailboxes, *entry.Mailbox)

		case addressbook.Forward, addressbook.Remote:
			queue = append(queue, entry.Address)
		}
	}

	log := log.WithField("mail", id)

	if len(mailboxes) > 0 {
		if err := m.DB.AddEntries(id, mailboxes); err != nil {
			return err
		}

		log.WithField("mailboxes", mailboxes).
			Debug("mail delivered to local mailboxes")
	}

	if len(queue) > 0 {
		if err := m.DB.AddToQueue(id, queue); err != nil {
			return err
		}

		log.Debug("mail queued for outbound delivery")

		m.Queue.WakeUp()
	}

	return nil
}
