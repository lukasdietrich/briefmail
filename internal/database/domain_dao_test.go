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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestDomainDaoTestSuite(t *testing.T) {
	suite.Run(t, new(DomainDaoTestSuite))
}

type DomainDaoTestSuite struct {
	suite.Suite

	ctx       context.Context
	conn      Conn
	domainDao DomainDao
}

func (s *DomainDaoTestSuite) SetupTest() {
	conn, err := openInMemory()
	s.Require().NoError(err)

	s.ctx = context.Background()
	s.conn = conn
	s.domainDao = NewDomainDao()
}

func (s *DomainDaoTestSuite) TearDownTest() {
	s.Require().NoError(s.conn.Close())
}

func (s *DomainDaoTestSuite) TestInsert() {
	domain := models.DomainEntity{
		Name: "new.example.com",
	}

	s.Assert().Zero(domain.ID)
	s.Assert().NoError(s.domainDao.Insert(s.ctx, s.conn, &domain))
	s.Assert().NotZero(domain.ID)

	domains, err := s.domainDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Assert().Equal([]models.DomainEntity{domain}, domains)
}

func (s *DomainDaoTestSuite) TestUpdate() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "domains"
				( "id", "name" )
			values
				( 42, 'outdated.example.com' ) ;
		`)

	domain := models.DomainEntity{
		ID:   42,
		Name: "updated.example.com",
	}

	s.Assert().NoError(s.domainDao.Update(s.ctx, s.conn, &domain))

	domains, err := s.domainDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Assert().Equal([]models.DomainEntity{domain}, domains)
}

func (s *DomainDaoTestSuite) TestDelete() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "domains"
				( "id", "name" )
			values
				( 42, 'wrong.example.com' ) ;
		`)

	domains, err := s.domainDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Require().Len(domains, 1)

	s.Assert().NoError(s.domainDao.Delete(s.ctx, s.conn, &models.DomainEntity{ID: 42}))

	domains, err = s.domainDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Assert().Len(domains, 0)
}

func (s *DomainDaoTestSuite) TestFindAll() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "domains"
				( "id", "name" )
			values
				( 42, 'c.example.com' ) ,
				( 43, 'a.example.com' ) ,
				( 44, 'b.example.com' ) ;
		`)

	expected := []models.DomainEntity{
		{ID: 43, Name: "a.example.com"},
		{ID: 44, Name: "b.example.com"},
		{ID: 42, Name: "c.example.com"},
	}

	actual, err := s.domainDao.FindAll(s.ctx, s.conn)
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}

func (s *DomainDaoTestSuite) TestFindByName() {
	s.conn.ExecContext(s.ctx,
		`
			insert into "domains"
				( "id", "name" )
			values
				( 42, 'c.example.com' ) ,
				( 43, 'a.example.com' ) ,
				( 44, 'b.example.com' ) ;
		`)

	expected := &models.DomainEntity{
		ID:   44,
		Name: "b.example.com",
	}

	actual, err := s.domainDao.FindByName(s.ctx, s.conn, "b.example.com")
	s.Assert().NoError(err)
	s.Assert().Equal(expected, actual)
}
