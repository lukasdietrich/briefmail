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
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/database"
	"github.com/lukasdietrich/briefmail/internal/models"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

func TestCleanerTestSuite(t *testing.T) {
	suite.Run(t, new(CleanerTestSuite))
}

type CleanerTestSuite struct {
	suite.Suite

	database *database.MockConn
	tx       *database.MockTx
	mailDao  *database.MockMailDao
	blobs    *storage.MockBlobs

	cleaner Cleaner
}

func (s *CleanerTestSuite) SetupTest() {
	s.database = new(database.MockConn)
	s.tx = new(database.MockTx)
	s.mailDao = new(database.MockMailDao)
	s.blobs = new(storage.MockBlobs)

	s.cleaner = NewCleaner(s.database, s.mailDao, s.blobs)
}

func (s *CleanerTestSuite) TeardownTest() {
	mock.AssertExpectationsForObjects(s.T(),
		s.database,
		s.tx,
		s.mailDao,
		s.blobs)
}

func (s *CleanerTestSuite) TestClean_beginTxError() {
	s.database.On("Begin", mock.Anything).Return(nil, errors.New("err1"))
	s.Assert().EqualError(s.cleaner.Clean(context.TODO()), "err1")
}

func (s *CleanerTestSuite) TestClean_findMailsError() {
	s.database.On("Begin", mock.Anything).Return(s.tx, nil)
	s.tx.On("Rollback").Return(nil)
	s.mailDao.On("FindDeletable", mock.Anything, s.tx).Return(nil, errors.New("err2"))

	s.Assert().EqualError(s.cleaner.Clean(context.TODO()), "err2")
}

func (s *CleanerTestSuite) TestClean_updateMailError() {
	s.database.On("Begin", mock.Anything).Return(s.tx, nil)
	s.tx.On("Rollback").Return(nil)
	s.mailDao.On("FindDeletable", mock.Anything, s.tx).Return([]models.MailEntity{{ID: "id3"}}, nil)
	s.mailDao.On("Update", mock.Anything, s.tx, mock.Anything).Return(errors.New("err3"))
	s.blobs.On("Delete", mock.Anything, "id3").Return(nil)

	s.Assert().EqualError(s.cleaner.Clean(context.TODO()), "err3")
}

func (s *CleanerTestSuite) TestClean_deleteBlobError() {
	s.database.On("Begin", mock.Anything).Return(s.tx, nil)
	s.tx.On("Rollback").Return(nil)
	s.mailDao.On("FindDeletable", mock.Anything, s.tx).Return([]models.MailEntity{{ID: "id4"}}, nil)
	s.blobs.On("Delete", mock.Anything, "id4").Return(errors.New("err4"))

	s.Assert().EqualError(s.cleaner.Clean(context.TODO()), "err4")
}

func (s *CleanerTestSuite) TestClean_ok() {
	mails := []models.MailEntity{
		{ID: "id1"},
		{ID: "id2"},
	}

	s.database.On("Begin", mock.Anything).Return(s.tx, nil)
	s.tx.On("Rollback").Return(nil)
	s.tx.On("Commit").Return(nil)
	s.mailDao.On("FindDeletable", mock.Anything, s.tx).Return(mails, nil)
	s.mailDao.On("Update",
		mock.Anything,
		s.tx,
		mock.MatchedBy(func(mail *models.MailEntity) bool {
			return mail.ID == "id1" || mail.ID == "id2"
		}),
	).Return(nil)
	s.blobs.On("Delete",
		mock.Anything,
		mock.MatchedBy(func(id string) bool {
			return id == "id1" || id == "id2"
		}),
	).Return(nil)

	s.Assert().NoError(s.cleaner.Clean(context.TODO()))
}
