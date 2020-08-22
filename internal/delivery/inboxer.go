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

package delivery

import (
	"context"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

// Inboxer is service to read a list of unread mails of a mailbox and committing changes later.
type Inboxer struct {
	database *storage.Database
	cleaner  *Cleaner
}

// NewInboxer creates a new Inboxer.
func NewInboxer(database *storage.Database, cleaner *Cleaner) *Inboxer {
	return &Inboxer{
		database: database,
		cleaner:  cleaner,
	}
}

// Inbox reads the a list of unread mails for a mailbox.
func (i *Inboxer) Inbox(ctx context.Context, mailbox *storage.Mailbox) (*Inbox, error) {
	tx, err := i.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	mails, err := queries.FindMailsByMailbox(tx, mailbox)
	if err != nil {
		return nil, err
	}

	log.InfoContext(ctx).
		Int64("mailbox", mailbox.ID).
		Int("mailCount", len(mails)).
		Msg("inbox loaded")

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return newInbox(mails), nil
}

// Commit removes all marked mails of an inbox from the mailbox.
func (i *Inboxer) Commit(ctx context.Context, mailbox *storage.Mailbox, inbox *Inbox) error {
	tx, err := i.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	log.InfoContext(ctx).
		Int64("mailbox", mailbox.ID).
		Msg("committing inbox changes")

	for index, mail := range inbox.Mails {
		if inbox.IsMarked(index) {
			log.DebugContext(ctx).
				Int64("mailbox", mailbox.ID).
				Str("mail", mail.ID).
				Msg("deleting mail from mailbox")

			if err := queries.DeleteMailboxEntry(tx, mailbox, &mail); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := i.cleaner.Clean(ctx); err != nil {
		log.ErrorContext(ctx).
			Err(err).
			Msg("error during cleanup")
	}

	return nil
}

// Inbox is a list of unreal mails as well as a set of "marks".
// Marked mails are removed, when the inbox state is committed.
type Inbox struct {
	Mails      []storage.Mail
	marks      map[int]bool
	size       int64
	sizeMarked int64
}

func newInbox(mails []storage.Mail) *Inbox {
	var totalSize int64
	for _, mail := range mails {
		totalSize += mail.Size
	}

	inbox := Inbox{
		Mails: mails,
		size:  totalSize,
	}

	inbox.Reset()
	return &inbox
}

// IsMarked checks if a mail is marked for removal.
func (i *Inbox) IsMarked(index int) bool {
	return i.marks[index]
}

// Mark marks a mail for removal.
func (i *Inbox) Mark(index int) {
	i.marks[index] = true
	i.sizeMarked += i.Mails[index].Size
}

// Reset removes all marks.
func (i *Inbox) Reset() {
	i.marks = make(map[int]bool)
	i.sizeMarked = 0
}

// Size is the sum of the sizes of non-marked mails.
func (i *Inbox) Size() int64 {
	return i.size - i.sizeMarked
}

// Count is the amount of non-marked mails.
func (i *Inbox) Count() int {
	return len(i.Mails) - len(i.marks)
}
