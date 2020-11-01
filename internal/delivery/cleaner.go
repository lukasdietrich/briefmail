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
	"time"

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

// Cleaner is a service to clean orphaned mail blobs and their database counterparts.
type Cleaner struct {
	database database.Conn
	mailDao  database.MailDao
	blobs    *storage.Blobs
}

// NewCleaner creates a new Cleaner.
func NewCleaner(db database.Conn, mailDao database.MailDao, blobs *storage.Blobs) *Cleaner {
	return &Cleaner{
		database: db,
		mailDao:  mailDao,
		blobs:    blobs,
	}
}

// Clean finds all orphaned mails and deletes them. An orphaned mail is a mail not assigned to a
// mailbox and not queued for outbound delivery.
func (c *Cleaner) Clean(ctx context.Context) error {
	tx, err := c.database.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	mails, err := c.mailDao.FindDeletable(ctx, tx)
	if err != nil {
		return err
	}

	for _, mail := range mails {
		if err := c.deleteMail(ctx, tx, &mail); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *Cleaner) deleteMail(ctx context.Context, tx database.Tx, mail *models.MailEntity) error {
	if err := c.blobs.Delete(ctx, mail.ID); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	mail.DeletedAt.Int64 = time.Now().Unix()
	mail.DeletedAt.Valid = true

	return c.mailDao.Update(ctx, tx, mail)
}
