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
	"time"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/crypto"
	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
)

var (
	// ErrWrongAddressPassword is returned when an address either does not exist or the password
	// does not match the hash.
	ErrWrongAddressPassword = errors.New("wrong address or password combination")
)

func init() {
	viper.SetDefault("security.auth.minDuration", "5s")
}

// Authenticator is for authentication of users based on their addresses.
type Authenticator interface {
	// Auth searches for a mailbox by address. If the address does not exist, is not local or the
	// password does not match the stored hash, ErrWrongAddressPassword is returned. Database errors
	// may occur.
	Auth(ctx context.Context, name, pass []byte) (*models.MailboxEntity, error)
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator(
	db database.Conn,
	mailboxCredentialDao database.MailboxCredentialDao,
	addressbook Addressbook,
) Authenticator {
	return &authenticator{
		database:             db,
		mailboxCredentialDao: mailboxCredentialDao,
		addressbook:          addressbook,

		minDuration: viper.GetDuration("security.auth.minDuration"),
	}
}

type authenticator struct {
	database             database.Conn
	mailboxCredentialDao database.MailboxCredentialDao
	addressbook          Addressbook

	minDuration time.Duration
}

func (a *authenticator) Auth(ctx context.Context, name, pass []byte) (*models.MailboxEntity, error) {
	startTime := time.Now()
	defer a.ensureMinDuration(startTime)

	result, err := a.lookup(ctx, name)
	if err != nil {
		return nil, err
	}

	if err := crypto.Verify(result.credentials, pass); err != nil {
		if errors.Is(err, crypto.ErrPasswordMismatch) {
			log.WarnContext(ctx).
				Bytes("name", name).
				Msg("failed auth attempt: wrong password")

			return nil, ErrWrongAddressPassword
		}

		return nil, err
	}

	return result.mailbox, nil
}

func (a *authenticator) ensureMinDuration(start time.Time) {
	elapsed := time.Since(start)
	remaining := a.minDuration - elapsed

	if remaining > 0 {
		time.Sleep(remaining)
	}
}

type mailboxWithCredentials struct {
	mailbox     *models.MailboxEntity
	credentials *models.MailboxCredentialEntity
}

func (a *authenticator) lookup(ctx context.Context, name []byte) (*mailboxWithCredentials, error) {
	addr, err := models.ParseNormalized(string(name))
	if err != nil {
		log.WarnContext(ctx).
			Bytes("name", name).
			Err(err).
			Msg("failed auth attempt: invalid address")

		return nil, ErrWrongAddressPassword
	}

	result, err := a.addressbook.Lookup(ctx, addr)
	if err != nil {
		return nil, err
	}

	if !result.IsLocal || result.Mailbox == nil {
		log.WarnContext(ctx).
			Bytes("name", name).
			Msg("failed auth attempt: unknown address")

		return nil, ErrWrongAddressPassword
	}

	credentials, err := a.mailboxCredentialDao.FindByMailbox(ctx, a.database, result.Mailbox)
	if err != nil {
		return nil, err
	}

	return &mailboxWithCredentials{result.Mailbox, credentials}, nil

}
