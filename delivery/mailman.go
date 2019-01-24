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

	"github.com/lukasdietrich/briefmail/addressbook"
	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/storage"
)

type Mailman struct {
	config *Config
}

type Config struct {
	DB    *storage.DB
	Blobs *storage.Blobs
	Book  *addressbook.Book
}

func NewMailman(config *Config) *Mailman {
	return &Mailman{config: config}
}

func (m *Mailman) Deliver(envelope *model.Envelope, mail model.Body) error {
	offset := mail.Prepend("Return-Path", fmt.Sprintf("<%s>", envelope.From))
	id, size, err := m.config.Blobs.Write(mail)
	if err != nil {
		return err
	}

	if err := m.config.DB.AddMail(id, size, offset, envelope); err != nil {
		m.config.Blobs.Delete(id)
		return err
	}

	var (
		mailboxes []int64
		queue     []*model.Address
	)

	for _, addr := range envelope.To {
		entry := m.config.Book.Lookup(addr)
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

	if len(queue) > 0 {
		if err := m.config.DB.AddToQueue(id, queue); err != nil {
			return err
		}
	}

	if len(mailboxes) > 0 {
		if err := m.config.DB.AddEntries(id, mailboxes); err != nil {
			return err
		}
	}

	return nil
}
