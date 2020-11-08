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

func TestMailboxDaoTestSuite(t *testing.T) {
	suite.Run(t, new(MailboxDaoTestSuite))
}

type MailboxDaoTestSuite struct {
	baseDatabaseTestSuite

	mailboxDao MailboxDao
}

func (s *MailboxDaoTestSuite) SetupSuite() {
	s.mailboxDao = NewMailboxDao()
}

func (s *MailboxDaoTestSuite) TestInsert() {
	mailbox := models.MailboxEntity{
		DisplayName: "test",
	}

	s.Assert().Zero(mailbox.ID)
	s.Assert().NoError(s.mailboxDao.Insert(s.ctx, s.conn, &mailbox))
	s.Assert().NotZero(mailbox.ID)

	s.assertQuery(
		`
			select "id", "display_name"
			from "mailboxes" ;
		`,
		[]string{"1", "test"})
}

func (s *MailboxDaoTestSuite) TestUpdate() {
	s.requireExec(
		`
			insert into "mailboxes" ( "id", "display_name" ) values ( 42, 'old-name' ) ;
		`)

	mailbox := models.MailboxEntity{
		ID:          42,
		DisplayName: "new-name",
	}

	s.Assert().NoError(s.mailboxDao.Update(s.ctx, s.conn, &mailbox))

	s.assertQuery(
		`
			select "id", "display_name"
			from "mailboxes" ;
		`,
		[]string{"42", "new-name"})
}

func (s *MailboxDaoTestSuite) TestDelete() {
	s.requireExec(
		`
			insert into "mailboxes" ( "id", "display_name" ) values ( 42, 'old-name' ) ;
		`)

	s.assertQuery(`select count(*) from "mailboxes" ;`, []string{"1"})
	s.Assert().NoError(s.mailboxDao.Delete(s.ctx, s.conn, &models.MailboxEntity{ID: 42}))
	s.assertQuery(`select count(*) from "mailboxes" ;`, []string{"0"})
}

func (s *MailboxDaoTestSuite) TestFindAll() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox1' ) ,
				( 43, 'mailbox2' ) ,
				( 44, 'mailbox3' ) ;
		`)

	expected := []models.MailboxEntity{
		{
			ID:          42,
			DisplayName: "mailbox1",
		},
		{
			ID:          43,
			DisplayName: "mailbox2",
		},
		{
			ID:          44,
			DisplayName: "mailbox3",
		},
	}

	actual, err := s.mailboxDao.FindAll(s.ctx, s.conn)
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *MailboxDaoTestSuite) TestFindByAddress() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox1' ) ,
				( 43, 'mailbox2' ) ;

			insert into "domains" ( "id", "name" ) values ( 420, 'example.com' ) ;

			insert into "addresses"
				( "id", "local_part", "domain_id", "mailbox_id" )
			values 
				( 123, 'addr1', 420, 42 ) ,
				( 124, 'addr2', 420, 43 ) ;
		`)

	expected := &models.MailboxEntity{
		ID:          43,
		DisplayName: "mailbox2",
	}

	actual, err := s.mailboxDao.FindByAddress(s.ctx, s.conn, s.mustParseAddress("addr2@example.com"))
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}
