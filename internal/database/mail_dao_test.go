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

func TestMailDaoTestSuite(t *testing.T) {
	suite.Run(t, new(MailDaoTestSuite))
}

type MailDaoTestSuite struct {
	baseDatabaseTestSuite

	mailDao MailDao
}

func (s *MailDaoTestSuite) SetupSuite() {
	s.mailDao = NewMailDao()
}

func (s *MailDaoTestSuite) TestInsert() {
	mail := models.MailEntity{
		ID:         "super-random-id",
		ReceivedAt: 42,
		DeletedAt: sql.NullInt64{
			Int64: 43,
			Valid: true,
		},
		ReturnPath:   s.mustParseAddress("someone@example.com"),
		Size:         44,
		AttemptCount: 45,
		LastAttemptedAt: sql.NullInt64{
			Int64: 46,
			Valid: true,
		},
	}

	s.Require().NoError(s.mailDao.Insert(s.ctx, s.conn, &mail))
	s.assertQuery(
		`
			select 
				"id", 
				"received_at", 
				"deleted_at", 
				"return_path", 
				"size", 
				"attempt_count", 
				"last_attempted_at"
			from "mails" ;
		`,
		[]string{"super-random-id", "42", "43", "someone@example.com", "44", "45", "46"})
}

func (s *MailDaoTestSuite) TestUpdate() {
	s.requireExec(
		`
			insert into "mails" (
				"id", 
				"received_at", 
				"deleted_at", 
				"return_path", 
				"size", 
				"attempt_count", 
				"last_attempted_at"
			) values 
				( 'real-mail', 1, null, 'old@example.com', 2, 3, null ) ;
		`)

	mail := models.MailEntity{
		ID:         "real-mail",
		ReceivedAt: 420,
		DeletedAt: sql.NullInt64{
			Int64: 421,
			Valid: true,
		},
		ReturnPath:   s.mustParseAddress("new@example.com"),
		Size:         422,
		AttemptCount: 423,
		LastAttemptedAt: sql.NullInt64{
			Int64: 424,
			Valid: true,
		},
	}

	s.Assert().NoError(s.mailDao.Update(s.ctx, s.conn, &mail))
	s.assertQuery(
		`
			select 
				"id", 
				"received_at", 
				"deleted_at", 
				"return_path", 
				"size", 
				"attempt_count", 
				"last_attempted_at"
			from "mails" ;
		`,
		[]string{"real-mail", "420", "421", "new@example.com", "422", "423", "424"})
}

func (s *MailDaoTestSuite) TestFindByMailbox() {
	s.requireExec(
		`
			insert into "mailboxes"
				( "id", "display_name" )
			values
				( 42, 'mailbox1' ) ,
				( 43, 'mailbox2' ) ;

			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count" ) 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 0 ) ,
				( 'mail2', 1338, 'someone2@example.com', 421, 0 ) ,
				( 'mail3', 1339, 'someone3@example.com', 422, 0 ) ;

			insert into "recipients"
				( "mail_id", "mailbox_id", "forward_path", "status" )
			values
				( 'mail1', null, 'somewhere@external', 4 ) ,
				( 'mail1', 42, 'someone@local', 4 ) ,
				( 'mail1', 43, 'someone2@local', 3 ) ,
				( 'mail2', 42, 'someone@local', 3 ) ,
				( 'mail3', 42, 'someone@local', 3 ) ,
				( 'mail3', 42, 'duplicate@local', 3 ) ;
		`)

	expected := []models.MailEntity{
		{
			ID:         "mail2",
			ReceivedAt: 1338,
			ReturnPath: s.mustParseAddress("someone2@example.com"),
			Size:       421,
		},
		{
			ID:         "mail3",
			ReceivedAt: 1339,
			ReturnPath: s.mustParseAddress("someone3@example.com"),
			Size:       422,
		},
	}

	actual, err := s.mailDao.FindByMailbox(s.ctx, s.conn, &models.MailboxEntity{ID: 42})
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *MailDaoTestSuite) TestFindDeleteable() {
	s.requireExec(
		`
			insert into "mails" 
				( "id", "received_at", "deleted_at", "return_path", "size", "attempt_count") 
			values 
				( 'mail1', 1337, 7331, 'someone1@example.com', 420, 0  ) ,
				( 'mail2', 1338, null, 'someone2@example.com', 421, 0  ) ,
				( 'mail3', 1339, null, 'someone3@example.com', 422, 0  ) ,
				( 'mail4', 1340, null, 'someone4@example.com', 423, 0  ) ;

			insert into "recipients"
				( "mail_id", "forward_path", "status" )
			values
				( 'mail2', 'somewhere@external', 3 ) ,
				( 'mail3', 'somewhere@external', 4 ) ,
				( 'mail4', 'somewhere1@external', 1 ) ,
				( 'mail4', 'somewhere2@external', 2 ) ;
		`)

	expected := []models.MailEntity{
		{
			ID:         "mail4",
			ReceivedAt: 1340,
			ReturnPath: s.mustParseAddress("someone4@example.com"),
			Size:       423,
		},
	}

	actual, err := s.mailDao.FindDeletable(s.ctx, s.conn)
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *MailDaoTestSuite) TestFindNextPending() {
	s.requireExec(
		`
			insert into "mails" 
				( "id", "received_at", "return_path", "size", "attempt_count", "last_attempted_at" ) 
			values 
				( 'mail1', 1337, 'someone1@example.com', 420, 3, 123 ) ,
				( 'mail2', 1338, 'someone2@example.com', 421, 2, 124 ) ,
				( 'mail3', 1339, 'someone3@example.com', 422, 3, 125 ) ;

			insert into "recipients"
				( "mail_id", "forward_path", "status" )
			values
				( 'mail2', 'somewhere@external', 4 ) ,
				( 'mail3', 'somewhere@external', 4 ) ;
		`)

	expected := &models.MailEntity{
		ID:           "mail2",
		ReceivedAt:   1338,
		ReturnPath:   s.mustParseAddress("someone2@example.com"),
		Size:         421,
		AttemptCount: 2,
		LastAttemptedAt: sql.NullInt64{
			Valid: true,
			Int64: 124,
		},
	}

	actual, err := s.mailDao.FindNextPending(s.ctx, s.conn)
	s.Require().NoError(err)
	s.Assert().Equal(expected, actual)
}
