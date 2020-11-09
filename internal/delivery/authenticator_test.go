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
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestAuthenticatorTestSuite(t *testing.T) {
	suite.Run(t, new(AuthenticatorTestSuite))
}

type AuthenticatorTestSuite struct {
	suite.Suite

	authenticator Authenticator

	db                   *database.MockConn
	mailboxCredentialDao *database.MockMailboxCredentialDao
	addressbook          *MockAddressbook
}

func (s *AuthenticatorTestSuite) SetupTest() {
	viper.Set("security.auth.minDuration", "0")

	s.db = new(database.MockConn)
	s.mailboxCredentialDao = new(database.MockMailboxCredentialDao)
	s.addressbook = new(MockAddressbook)

	s.authenticator = NewAuthenticator(s.db, s.mailboxCredentialDao, s.addressbook)
}

func (s *AuthenticatorTestSuite) TeardownTest() {
	mock.AssertExpectationsForObjects(s.T(),
		s.db,
		s.mailboxCredentialDao,
		s.addressbook)
}

func (s *AuthenticatorTestSuite) TestAuthSuccessful() {
	ctx := log.WithCommand(context.TODO(), "TestAuthSuccessful")
	expected := &models.MailboxEntity{ID: 42}

	s.addressbook.
		On("Lookup",
			ctx,
			mock.MatchedBy(func(addr models.Address) bool {
				return addr.String() == "someone@example.com"
			}),
		).
		Return(
			&LookupResult{
				IsLocal: true,
				Mailbox: expected,
			},
			nil,
		)

	s.mailboxCredentialDao.On("FindByMailbox", ctx, s.db, expected).Return(
		&models.MailboxCredentialEntity{
			Hash: "$argon2id$v=19$m=65536,t=2,p=4$2J03Yrz/w1hXkWm3WklAwg$pucJv/13J9vzUJEvIKwUi7INRi+dkNFigdC2CRHTr+k",
		},
		nil,
	)

	actual, err := s.authenticator.Auth(ctx, []byte("someone+suffix@example.com"), []byte("hunter2"))
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *AuthenticatorTestSuite) TestAuthWrongPassword() {
	ctx := log.WithCommand(context.TODO(), "TestAuthWrongPassword")
	mailbox := &models.MailboxEntity{ID: 42}

	s.addressbook.
		On("Lookup",
			ctx,
			mock.MatchedBy(func(addr models.Address) bool {
				return addr.String() == "someone@example.com"
			}),
		).
		Return(
			&LookupResult{
				IsLocal: true,
				Mailbox: mailbox,
			},
			nil,
		)

	s.mailboxCredentialDao.On("FindByMailbox", ctx, s.db, mailbox).Return(
		&models.MailboxCredentialEntity{
			Hash: "$argon2id$v=19$m=65536,t=2,p=4$2J03Yrz/w1hXkWm3WklAwg$pucJv/13J9vzUJEvIKwUi7INRi+dkNFigdC2CRHTr+k",
		},
		nil,
	)

	actual, err := s.authenticator.Auth(ctx, []byte("someone+suffix@example.com"), []byte("hunter3"))
	s.Assert().Nil(actual)
	s.Assert().Equal(ErrWrongAddressPassword, err)
}

func (s *AuthenticatorTestSuite) TestAuthUnknownAddress() {
	ctx := log.WithCommand(context.TODO(), "TestAuthUnknownAddress")

	s.addressbook.On("Lookup", ctx, mock.Anything).Return(
		&LookupResult{
			IsLocal: true,
			Mailbox: nil,
		},
		nil,
	)

	actual, err := s.authenticator.Auth(ctx, []byte("someone+suffix@example.com"), []byte("hunter2"))
	s.Assert().Nil(actual)
	s.Assert().Equal(ErrWrongAddressPassword, err)
}

func (s *AuthenticatorTestSuite) TestAuthMinDurationFaster() {
	viper.Set("security.auth.minDuration", "100ms")
	s.authenticator = NewAuthenticator(s.db, s.mailboxCredentialDao, s.addressbook)

	ctx := log.WithCommand(context.TODO(), "TestAuthMinDurationFaster")

	s.addressbook.On("Lookup", ctx, mock.Anything).Return(
		&LookupResult{
			IsLocal: true,
			Mailbox: nil,
		},
		nil,
	)

	startTime := time.Now()
	expectedEndTime := startTime.Add(100 * time.Millisecond)

	actual, err := s.authenticator.Auth(ctx, []byte("someone+suffix@example.com"), []byte("hunter2"))
	s.Assert().Nil(actual)
	s.Assert().Equal(ErrWrongAddressPassword, err)

	s.Assert().WithinDuration(expectedEndTime, time.Now(), 25*time.Millisecond)
}

func (s *AuthenticatorTestSuite) TestAuthMinDurationSlower() {
	viper.Set("security.auth.minDuration", "100ms")
	s.authenticator = NewAuthenticator(s.db, s.mailboxCredentialDao, s.addressbook)

	ctx := log.WithCommand(context.TODO(), "TestAuthMinDurationSlower")

	s.addressbook.On("Lookup", ctx, mock.Anything).After(200*time.Millisecond).Return(
		&LookupResult{
			IsLocal: true,
			Mailbox: nil,
		},
		nil,
	)

	startTime := time.Now()
	expectedEndTime := startTime.Add(200 * time.Millisecond)

	actual, err := s.authenticator.Auth(ctx, []byte("someone+suffix@example.com"), []byte("hunter2"))
	s.Assert().Nil(actual)
	s.Assert().Equal(ErrWrongAddressPassword, err)

	s.Assert().WithinDuration(expectedEndTime, time.Now(), 25*time.Millisecond)
}
