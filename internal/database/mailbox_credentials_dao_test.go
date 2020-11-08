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

func TestMailboxCredentialDaoTestSuite(t *testing.T) {
	suite.Run(t, new(MailboxCredentialDaoTestSuite))
}

type MailboxCredentialDaoTestSuite struct {
	baseDatabaseTestSuite

	mailboxCredentialDao MailboxCredentialDao
}

func (s *MailboxCredentialDaoTestSuite) SetupSuite() {
	s.mailboxCredentialDao = NewMailboxCredentialDao()
}

func (s *MailboxCredentialDaoTestSuite) TestUpsert_newEntry() {
	s.requireExec(
		`
			insert into "mailboxes" ( "id", "display_name" ) values ( 123, 'Someone' ) ;
		`)

	credentials := models.MailboxCredentialEntity{
		MailboxID: 123,
		UpdatedAt: 1337,
		Hash:      "super-secret-hash",
	}

	s.Require().NoError(s.mailboxCredentialDao.Upsert(s.ctx, s.conn, &credentials))

	s.assertQuery(
		`
			select "mailbox_id", "updated_at", "hash"
			from "mailbox_credentials" ;
		`,
		[]string{"123", "1337", "super-secret-hash"})
}

func (s *MailboxCredentialDaoTestSuite) TestUpsert_overwriteEntry() {
	s.requireExec(
		`
			insert into "mailboxes" ( "id", "display_name" ) values ( 123, 'Someone' ) ;

			insert into "mailbox_credentials"
				( "mailbox_id", "updated_at", "hash" )
			values
				( 123, 42, 'not-so-secret' ) ;
		`)

	credentials := models.MailboxCredentialEntity{
		MailboxID: 123,
		UpdatedAt: 1337,
		Hash:      "super-secret-hash",
	}

	s.Require().NoError(s.mailboxCredentialDao.Upsert(s.ctx, s.conn, &credentials))

	s.assertQuery(
		`
			select "mailbox_id", "updated_at", "hash"
			from "mailbox_credentials" ;
		`,
		[]string{"123", "1337", "super-secret-hash"})
}

func (s *MailboxCredentialDaoTestSuite) TestFindByMailbox() {
	s.requireExec(
		`
			insert into "mailboxes" ( "id", "display_name" ) values ( 123, 'Someone' ) ;

			insert into "mailbox_credentials"
				( "mailbox_id", "updated_at", "hash" )
			values
				( 123, 42, 'not-so-secret' ) ;
		`)

	mailbox := models.MailboxEntity{ID: 123}

	expected := models.MailboxCredentialEntity{
		MailboxID: 123,
		UpdatedAt: 42,
		Hash:      "not-so-secret",
	}

	actual, err := s.mailboxCredentialDao.FindByMailbox(s.ctx, s.conn, &mailbox)
	s.Assert().NoError(err)
	s.Assert().Equal(&expected, actual)
}
