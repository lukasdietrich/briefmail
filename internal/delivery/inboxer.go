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

	"github.com/lukasdietrich/briefmail/internal/storage"
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
	return nil, nil
}

// Commit removes all marked mails of an inbox from the mailbox.
func (i *Inboxer) Commit(ctx context.Context, mailbox *storage.Mailbox, inbox *Inbox) error {
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

// IsMarked checks if a mail is marked for removal.
func (i *Inbox) IsMarked(index int) bool {
	return i.marks[index]
}

// Mark marks a mail for removal.
func (i *Inbox) Mark(index int) {
	i.marks[index] = true
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
