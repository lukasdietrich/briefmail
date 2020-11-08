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
	"context"
	"database/sql/driver"

	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/models"
)

type baseDatabaseTestSuite struct {
	suite.Suite

	ctx  context.Context
	conn Conn
}

func (s *baseDatabaseTestSuite) SetupTest() {
	conn, err := openInMemory()
	s.Require().NoError(err)

	s.ctx = context.Background()
	s.conn = conn
}

func (s *baseDatabaseTestSuite) TearDownTest() {
	s.Require().NoError(s.conn.Close())
}

func (s *baseDatabaseTestSuite) requireExec(query string) {
	_, err := s.conn.ExecContext(s.ctx, query)
	s.Require().NoError(err)
}

func (s *baseDatabaseTestSuite) assertQuery(query string, expectedRows ...[]string) {
	rows, err := s.conn.QueryxContext(s.ctx, query)
	s.Require().NoError(err)

	defer rows.Close()

	for _, expectedValues := range expectedRows {
		s.Require().True(rows.Next())

		actualValues, err := rows.SliceScan()
		s.Require().NoError(err)
		s.Require().Len(actualValues, len(expectedValues))

		for i, actualValue := range actualValues {
			actualValueAsString, err := driver.String.ConvertValue(actualValue)
			s.Assert().NoError(err)
			s.Assert().Equal(expectedValues[i], actualValueAsString)
		}
	}

	s.Assert().False(rows.Next())
}

func (s *baseDatabaseTestSuite) mustParseAddress(raw string) models.Address {
	addr, err := models.Parse(raw)
	s.Require().NoError(err)
	return addr
}
