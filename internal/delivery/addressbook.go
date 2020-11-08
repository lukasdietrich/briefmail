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

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/models"
)

// LookupResult is the result of an address lookup.
type LookupResult struct {
	// Address is the address used for lookup.
	Address models.Address
	// IsLocal indicates if the domain part of the address is local. This does not imply that the
	// address exists.
	IsLocal bool
	// Mailbox is the local mailbox of an address, if it is local and exists. If Mailbox is not nil
	// IsLocal is implied to be true.
	Mailbox *models.MailboxEntity
}

// Addressbook is a registry to lookup mail addresses.
type Addressbook interface {
	// Lookup looks up an address without a transaction. See LookupTx.
	Lookup(context.Context, models.Address) (*LookupResult, error)
	// LookupTx looks up an address. The result indicates if the address belongs to a local domain
	// and if it does, if it exists. Only database errors may occur.
	LookupTx(context.Context, database.Queryer, models.Address) (*LookupResult, error)
}

// NewAddressbook creates a new Addressbook.
func NewAddressbook(
	db database.Conn,
	domainDao database.DomainDao,
	mailboxDao database.MailboxDao,
) Addressbook {
	return &addressbook{
		database:   db,
		domainDao:  domainDao,
		mailboxDao: mailboxDao,
	}
}

type addressbook struct {
	database   database.Conn
	domainDao  database.DomainDao
	mailboxDao database.MailboxDao
}

func (a *addressbook) Lookup(ctx context.Context, recipient models.Address) (*LookupResult, error) {
	return a.LookupTx(ctx, a.database, recipient)
}

func (a *addressbook) LookupTx(
	ctx context.Context,
	q database.Queryer,
	recipient models.Address,
) (*LookupResult, error) {
	isLocal, err := a.checkLocal(ctx, q, recipient)
	if err != nil {
		return nil, err
	}

	if !isLocal {
		return &LookupResult{Address: recipient, IsLocal: false}, nil
	}

	mailbox, err := a.mailboxDao.FindByAddress(ctx, q, recipient.Normalized())
	if err != nil && !database.IsErrNoRows(err) {
		return nil, err
	}

	return &LookupResult{Address: recipient, IsLocal: isLocal, Mailbox: mailbox}, nil
}

func (a *addressbook) checkLocal(
	ctx context.Context,
	q database.Queryer,
	address models.Address,
) (bool, error) {
	_, err := a.domainDao.FindByName(ctx, q, address.Domain())
	if err != nil {
		if database.IsErrNoRows(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
