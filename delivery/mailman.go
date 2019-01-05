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
	"strings"

	"github.com/lukasdietrich/briefmail/addressbook"
	"github.com/lukasdietrich/briefmail/model"
	"github.com/lukasdietrich/briefmail/storage"
)

type Mailman struct {
	config  *Config
	domains map[string]bool
}

type Config struct {
	DB           *storage.DB
	Blobs        *storage.Blobs
	LocalDomains []string
	Book         *addressbook.Book
}

func NewMailman(config *Config) *Mailman {
	domains := make(map[string]bool)
	for _, domain := range config.LocalDomains {
		domains[strings.ToLower(domain)] = true
	}

	return &Mailman{
		config:  config,
		domains: domains,
	}
}

func (m *Mailman) IsDeliverable(address *model.Address, local bool) bool {
	if m.domains[address.Domain] {
		_, ok := m.config.Book.Lookup(address)
		return ok
	}

	return !local
}

func (m *Mailman) Deliver(envelope *model.Envelope, mail io.Reader) error {
	id, size, err := m.config.Blobs.Write(mail)
	if err != nil {
		return err
	}

	if err := m.config.DB.AddMail(id, size, envelope); err != nil {
		m.config.Blobs.Delete(id)
		return err
	}

	local, remote := m.partitionRecipients(envelope)

	if len(remote) > 0 {
		if err := m.config.DB.AddToQueue(id, remote); err != nil {
			return err
		}
	}

	if len(local) > 0 {
		var mailboxes []int64

		for _, address := range local {
			entry, ok := m.config.Book.Lookup(address)
			if !ok {
				return fmt.Errorf("could not deliver to %s", address)
			}

			switch entry.Kind {
			case addressbook.Local:
				mailboxes = append(mailboxes, *entry.Mailbox)
			}
		}

		if len(mailboxes) > 0 {
			if err := m.config.DB.AddEntries(id, mailboxes); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Mailman) partitionRecipients(envelope *model.Envelope) (
	local []*model.Address,
	remote []*model.Address,
) {
	for _, to := range envelope.To {
		if m.domains[to.Domain] {
			local = append(local, to)
		} else {
			remote = append(remote, to)
		}
	}

	return
}
