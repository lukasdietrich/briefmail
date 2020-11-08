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
	"database/sql"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestRecipientDaoTestSuite(t *testing.T) {
	suite.Run(t, new(RecipientDaoTestSuite))
}

type RecipientDaoTestSuite struct {
	baseDatabaseTestSuite

	recipientDao RecipientDao
}

func (s *RecipientDaoTestSuite) SetupSuite() {
	s.recipientDao = NewRecipientDao()
}

func (s *RecipientDaoTestSuite) TestInsert() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 123, 'mailbox123' ) ;

			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count") 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 0  ) ;
		`)

	recipient := models.RecipientEntity{
		MailID: "mail1",
		MailboxID: sql.NullInt64{
			Int64: 123,
			Valid: true,
		},
		ForwardPath: s.mustParseAddress("someone@example.com"),
		Status:      3,
	}

	s.Assert().Zero(recipient.ID)
	s.Assert().NoError(s.recipientDao.Insert(s.ctx, s.conn, &recipient))
	s.Assert().NotZero(recipient.ID)

	s.assertQuery(
		`
			select "id", "mail_id", "mailbox_id", "forward_path", "status"
			from "recipients" ;
		`,
		[]string{"1", "mail1", "123", "someone@example.com", "3"})
}

func (s *RecipientDaoTestSuite) TestUpdate() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox42' ) ,
				( 43, 'mailbox43' ) ;

			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count") 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 0  ) ,
				( 'mail2', 1337, 'someone1@example.com', 420, 0  ) ;

			insert into "recipients"
				( "id", "mail_id", "mailbox_id", "forward_path", "status" )
			values
				( 123, 'mail1', 42, 'someone@somewhere', 3 ) ;
		`)

	recipient := models.RecipientEntity{
		ID:     123,
		MailID: "mail2",
		MailboxID: sql.NullInt64{
			Valid: true,
			Int64: 43,
		},
		ForwardPath: s.mustParseAddress("another@example.com"),
		Status:      4,
	}

	s.Assert().NoError(s.recipientDao.Update(s.ctx, s.conn, &recipient))

	s.assertQuery(
		`
			select "id", "mail_id", "mailbox_id", "forward_path", "status"
			from "recipients" ;
		`,
		[]string{"123", "mail2", "43", "another@example.com", "4"})
}

func (s *RecipientDaoTestSuite) TestUpdateDelivered() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox42' ) ;

			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count") 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 0  ) ;

			insert into "recipients"
				( "id", "mail_id", "mailbox_id", "forward_path", "status" )
			values
				( 123, 'mail1', 42, 'someone1@somewhere', 4 ) ,
				( 124, 'mail1', 42, 'someone2@somewhere', 4 ) ,
				( 125, 'mail1', null, 'someone3@somewhere', 4 ) ;
		`)

	s.Assert().NoError(s.recipientDao.UpdateDelivered(s.ctx, s.conn,
		&models.MailboxEntity{ID: 42}, &models.MailEntity{ID: "mail1"}))

	s.assertQuery(
		`
			select "id", "mail_id", "mailbox_id", "forward_path", "status"
			from "recipients" ;
		`,
		[]string{"123", "mail1", "42", "someone1@somewhere", "2"},
		[]string{"124", "mail1", "42", "someone2@somewhere", "2"},
		[]string{"125", "mail1", "<nil>", "someone3@somewhere", "4"},
	)
}

func (s *RecipientDaoTestSuite) TestFindPending() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox42' ) ;

			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count") 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 0  ) ;

			insert into "recipients"
				( "id", "mail_id", "mailbox_id", "forward_path", "status" )
			values
				( 123, 'mail1', 42, 'someone1@somewhere', 4 ) ,
				( 124, 'mail1', 42, 'someone2@somewhere', 1 ) ,
				( 125, 'mail1', null, 'someone3@somewhere', 4 ) ;
		`)

	expected := []models.RecipientEntity{
		{
			ID:     123,
			MailID: "mail1",
			MailboxID: sql.NullInt64{
				Valid: true,
				Int64: 42,
			},
			ForwardPath: s.mustParseAddress("someone1@somewhere"),
			Status:      4,
		},
		{
			ID:          125,
			MailID:      "mail1",
			ForwardPath: s.mustParseAddress("someone3@somewhere"),
			Status:      4,
		},
	}

	actual, err := s.recipientDao.FindPending(s.ctx, s.conn, &models.MailEntity{ID: "mail1"})
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}
