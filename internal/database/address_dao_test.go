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

package database

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestAddressDaoTestSuite(t *testing.T) {
	suite.Run(t, new(AddressDaoTestSuite))
}

type AddressDaoTestSuite struct {
	baseDatabaseTestSuite

	addressDao AddressDao
}

func (s *AddressDaoTestSuite) SetupSuite() {
	s.addressDao = NewAddressDao()
}

func (s *AddressDaoTestSuite) TestInsert() {
	s.requireExec(
		`
			insert into "domains" ( "id", "name" ) values ( 42, 'example.com' ) ;
			insert into "mailboxes" ( "id", "display_name" ) values ( 1337, 'Someone' ) ;
		`)

	address := models.AddressEntity{
		LocalPart: "someone",
		DomainID:  42,
		MailboxID: 1337,
	}

	s.Assert().Zero(address.ID)
	s.Assert().NoError(s.addressDao.Insert(s.ctx, s.conn, &address))
	s.Assert().NotZero(address.ID)

	s.assertQuery(
		`
			select "id", "domain_id", "mailbox_id", "local_part"
			from "addresses" ;
		`,
		[]string{"1", "42", "1337", "someone"},
	)
}

func (s *AddressDaoTestSuite) TestDelete() {
	s.requireExec(
		`
			insert into "domains" ( "id", "name" ) values ( 42, 'example.com' ) ;
			insert into "mailboxes" ( "id", "display_name" ) values ( 1337, 'Someone' ) ;

			insert into "addresses"
				( "id", "local_part", "domain_id", "mailbox_id" )
			values
				( 123, 'someone', 42, 1337 ) ;
		`)

	s.assertQuery(`select count(*) from "addresses" ;`, []string{"1"})
	s.Assert().NoError(s.addressDao.Delete(s.ctx, s.conn, &models.AddressEntity{ID: 123}))
	s.assertQuery(`select count(*) from "addresses" ;`, []string{"0"})
}

func (s *AddressDaoTestSuite) TestFindAll() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 1337, 'Person1' ) ,
				( 1338, 'Person2' ) ;

			insert into "domains" 
				( "id", "name" )
			values
				( 42, 'a.example.com' ) ,
				( 43, 'b.example.com' ) ;

			insert into "addresses"
				( "id", "local_part", "domain_id", "mailbox_id" )
			values 
				( 123, 'addr1', 42, 1337 ) ,
				( 124, 'addr2', 43, 1337 ) ,
				( 125, 'addr3', 42, 1338 ) ,
				( 126, 'addr4', 43, 1338 ) ;
		`)

	expected := []AddressWithDomain{
		{
			AddressEntity: models.AddressEntity{
				ID:        123,
				LocalPart: "addr1",
				DomainID:  42,
				MailboxID: 1337,
			},
			DomainName: "a.example.com",
		},
		{
			AddressEntity: models.AddressEntity{
				ID:        124,
				LocalPart: "addr2",
				DomainID:  43,
				MailboxID: 1337,
			},
			DomainName: "b.example.com",
		},
		{
			AddressEntity: models.AddressEntity{
				ID:        125,
				LocalPart: "addr3",
				DomainID:  42,
				MailboxID: 1338,
			},
			DomainName: "a.example.com",
		},
		{
			AddressEntity: models.AddressEntity{
				ID:        126,
				LocalPart: "addr4",
				DomainID:  43,
				MailboxID: 1338,
			},
			DomainName: "b.example.com",
		},
	}

	actual, err := s.addressDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)

}

func (s *AddressDaoTestSuite) TestFindByMailbox() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 1337, 'Person1' ) ,
				( 1338, 'Person2' ) ;


			insert into "domains" 
				( "id", "name" )
			values
				( 42, 'a.example.com' ) ,
				( 43, 'b.example.com' ) ;

			insert into "addresses"
				( "id", "local_part", "domain_id", "mailbox_id" )
			values 
				( 123, 'addr1', 42, 1337 ) ,
				( 124, 'addr2', 43, 1337 ) ,
				( 125, 'addr3', 42, 1338 ) ,
				( 126, 'addr4', 43, 1338 ) ;
		`)

	expected := []AddressWithDomain{
		{
			AddressEntity: models.AddressEntity{
				ID:        123,
				LocalPart: "addr1",
				DomainID:  42,
				MailboxID: 1337,
			},
			DomainName: "a.example.com",
		},
		{
			AddressEntity: models.AddressEntity{
				ID:        124,
				LocalPart: "addr2",
				DomainID:  43,
				MailboxID: 1337,
			},
			DomainName: "b.example.com",
		},
	}

	actual, err := s.addressDao.FindByMailbox(s.ctx, s.conn, &models.MailboxEntity{ID: 1337})
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)

}
