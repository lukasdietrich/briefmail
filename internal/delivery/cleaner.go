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
	"errors"
	"os"

	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

// Cleaner is a service to clean orphaned mail blobs and their database counterparts.
type Cleaner struct {
	database *storage.Database
	blobs    *storage.Blobs
}

// NewCleaner creates a new Cleaner.
func NewCleaner(database *storage.Database, blobs *storage.Blobs) *Cleaner {
	return &Cleaner{
		database: database,
		blobs:    blobs,
	}
}

// Clean finds all orphaned mails and deletes them. An orphaned mail is a mail not assigned to a
// mailbox and not queued for outbound delivery.
func (c *Cleaner) Clean(ctx context.Context) error {
	tx, err := c.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	mails, err := queries.FindOrphanedMails(tx)
	if err != nil {
		return err
	}

	for _, mail := range mails {
		if err := c.deleteMail(tx, &mail); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *Cleaner) deleteMail(tx *storage.Tx, mail *storage.Mail) error {
	if err := c.blobs.Delete(mail.ID); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return queries.DeleteMail(tx, mail)
}
