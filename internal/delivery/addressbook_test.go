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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestAddressbookTestSuite(t *testing.T) {
	suite.Run(t, new(AddressbookTestSuite))
}

type AddressbookTestSuite struct {
	suite.Suite

	addressbook Addressbook

	db         *database.MockConn
	domainDao  *database.MockDomainDao
	mailboxDao *database.MockMailboxDao
}

func (s *AddressbookTestSuite) SetupTest() {
	s.db = new(database.MockConn)
	s.domainDao = new(database.MockDomainDao)
	s.mailboxDao = new(database.MockMailboxDao)

	s.addressbook = NewAddressbook(s.db, s.domainDao, s.mailboxDao)
}

func (s *AddressbookTestSuite) TeardownTest() {
	mock.AssertExpectationsForObjects(s.T(),
		s.db,
		s.domainDao,
		s.mailboxDao)
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
