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
	"database/sql"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"

	database "github.com/lukasdietrich/briefmail/internal/mocks/database"
)

func TestAddressbookTestSuite(t *testing.T) {
	suite.Run(t, new(AddressbookTestSuite))
}

type AddressbookTestSuite struct {
	suite.Suite

	addressbook Addressbook

	db         *database.Conn
	domainDao  *database.DomainDao
	mailboxDao *database.MailboxDao
}

func (s *AddressbookTestSuite) SetupTest() {
	s.db = new(database.Conn)
	s.domainDao = new(database.DomainDao)
	s.mailboxDao = new(database.MailboxDao)

	s.addressbook = NewAddressbook(s.db, s.domainDao, s.mailboxDao)
}

func (s *AddressbookTestSuite) TeardownTest() {
	s.db.AssertExpectations(s.T())
	s.domainDao.AssertExpectations(s.T())
	s.mailboxDao.AssertExpectations(s.T())
}

func (s *AddressbookTestSuite) TestLookupExistingAddress() {
	address, _ := models.Parse("someone@example.com")
	ctx := log.WithCommand(context.TODO(), "TestLookupExistingAddress")

	s.domainDao.On("FindByName", ctx, s.db, "example.com").Return(
		&models.DomainEntity{
			ID:   1,
			Name: "example.com",
		},
		nil,
	)

	s.mailboxDao.On("FindByAddress", ctx, s.db, address).Return(
		&models.MailboxEntity{
			ID:          2,
			DisplayName: "someone-mailbox",
		},
		nil,
	)

	expected := &LookupResult{
		Address: address,
		IsLocal: true,
		Mailbox: &models.MailboxEntity{
			ID:          2,
			DisplayName: "someone-mailbox",
		},
	}

	actual, err := s.addressbook.Lookup(ctx, address)
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *AddressbookTestSuite) TestLookupAddressNotLocal() {
	address, _ := models.Parse("someone@example.com")
	ctx := log.WithCommand(context.TODO(), "TestLookupAddressNotLocal")

	s.domainDao.On("FindByName", ctx, s.db, "example.com").Return(nil, sql.ErrNoRows)

	expected := &LookupResult{
		Address: address,
		IsLocal: false,
	}

	actual, err := s.addressbook.Lookup(ctx, address)
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *AddressbookTestSuite) TestLookupAddressNotFound() {
	address, _ := models.Parse("someone@example.com")
	ctx := log.WithCommand(context.TODO(), "TestLookupAddressNotFound")

	s.domainDao.On("FindByName", ctx, s.db, "example.com").Return(
		&models.DomainEntity{
			ID:   1,
			Name: "example.com",
		},
		nil,
	)

	s.mailboxDao.On("FindByAddress", ctx, s.db, address).Return(nil, sql.ErrNoRows)

	expected := &LookupResult{
		Address: address,
		IsLocal: true,
	}

	actual, err := s.addressbook.Lookup(ctx, address)
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}
