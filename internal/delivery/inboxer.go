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

type Inboxer struct {
	database *storage.Database
	cleaner  *Cleaner
}

func NewInboxer(database *storage.Database, cleaner *Cleaner) *Inboxer {
	return &Inboxer{
		database: database,
		cleaner:  cleaner,
	}
}

func (i *Inboxer) Inbox(ctx context.Context, mailbox *storage.Mailbox) (*Inbox, error) {
	return nil, nil
}

func (i *Inboxer) Commit(ctx context.Context, mailbox *storage.Mailbox, inbox *Inbox) error {
	return nil
}

type Inbox struct {
	Mails      []storage.Mail
	marks      map[int]bool
	size       int64
	sizeMarked int64
}

func (i *Inbox) IsMarked(index int) bool {
	return i.marks[index]
}

func (i *Inbox) Mark(index int) {
	i.marks[index] = true
}

func (i *Inbox) Reset() {
}

func (i *Inbox) Size() int64 {
	return i.size - i.sizeMarked
}

func (i *Inbox) Count() int {
	return len(i.Mails) - len(i.marks)
}
