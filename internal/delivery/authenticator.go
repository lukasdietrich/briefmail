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
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

var (
	// ErrWrongAddressPassword is returned when an address either does not exist or the password
	// does not match the hash.
	ErrWrongAddressPassword = errors.New("wrong address or password combination")
)

// Authenticator is for authentication of users based on their addresses.
type Authenticator struct {
	database *storage.Database
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(database *storage.Database) *Authenticator {
	return &Authenticator{
		database: database,
	}
}

// Auth searches for a mailbox by address. If the address does not exist, is not local or the
// password does not match the stored hash, ErrWrongAddressPassword is returned. Database errors
// may occur.
func (a *Authenticator) Auth(ctx context.Context, name, pass []byte) (*storage.Mailbox, error) {
	mailbox, err := a.lookup(ctx, name)
	if err != nil {
		if isErrUnknownAddress(err) {
			logrus.Infof("failed auth attempt for %q: unknown or invalid address", name)
			return nil, ErrWrongAddressPassword
		}

		return nil, err
	}

	if err := crypto.Verify(mailbox, pass); err != nil {
		if errors.Is(err, crypto.ErrPasswordMismatch) {
			logrus.Infof("failed auth attempt for %q: wrong password", name)
			return nil, ErrWrongAddressPassword
		}

		return nil, err
	}

	return mailbox, nil
}

func (a *Authenticator) lookup(ctx context.Context, name []byte) (*storage.Mailbox, error) {
	addr, err := mails.ParseNormalized(string(name))
	if err != nil {
		return nil, err
	}

	tx, err := a.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	mailbox, err := queries.FindMailboxByAddress(tx, addr.LocalPart(), addr.Domain())
	if err != nil {
		return nil, err
	}

	return mailbox, tx.Commit()
}

func isErrUnknownAddress(err error) bool {
	return storage.IsErrNoRows(err) ||
		errors.Is(err, mails.ErrInvalidAddressFormat) ||
		errors.Is(err, mails.ErrPathTooLong)
}