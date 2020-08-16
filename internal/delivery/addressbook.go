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

	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

// Addressbook is a registry to lookup mail addresses.
type Addressbook struct {
	database *storage.Database
}

// NewAddressbook creates a new Addressbook.
func NewAddressbook(database *storage.Database) *Addressbook {
	return &Addressbook{
		database: database,
	}
}

// LookupResult is the result of an address lookup.
type LookupResult struct {
	// IsLocal indicates if the domain part of the address is local. This does not imply that the
	// address exists.
	IsLocal bool
	// Mailbox is the local mailbox of an address, if it is local and exists. If Mailbox is not nil
	// IsLocal is implied to be true.
	Mailbox *storage.Mailbox
}

// Lookup looks up an address in a new transaction. See LookupTx.
func (a *Addressbook) Lookup(ctx context.Context, recipient mails.Address) (*LookupResult, error) {
	tx, err := a.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	result, err := a.LookupTx(tx, recipient)
	if err != nil {
		return nil, err
	}

	return result, tx.Commit()
}

// LookupTx looks up an address. The result indicates if the address belongs to a local domain and
// if it does, if it exists. Only database errors may occur.
func (a *Addressbook) LookupTx(tx *storage.Tx, recipient mails.Address) (*LookupResult, error) {
	domain := recipient.Domain()

	isLocal, err := queries.ExistsDomain(tx, domain)
	if err != nil {
		return nil, err
	}

	if !isLocal {
		return &LookupResult{IsLocal: false}, nil
	}

	localPart := recipient.LocalPart()
	localPart = mails.NormalizeLocalPart(localPart)

	mailbox, err := queries.FindMailboxByAddress(tx, localPart, domain)
	if err != nil && !storage.IsErrNoRows(err) {
		return nil, err
	}

	return &LookupResult{IsLocal: isLocal, Mailbox: mailbox}, nil
}
