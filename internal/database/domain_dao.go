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

	"github.com/lukasdietrich/briefmail/internal/models"
)

// DomainDao is a data access object for all domain related queries.
type DomainDao interface {
	// Insert inserts a new domain.
	Insert(context.Context, Queryer, *models.DomainEntity) error
	// Update updates an existing domain.
	Update(context.Context, Queryer, *models.DomainEntity) error
	// Delete deletes an existing domain.
	Delete(context.Context, Queryer, *models.DomainEntity) error
	// FindAll returns all domains sorted by name.
	FindAll(context.Context, Queryer) ([]models.DomainEntity, error)
	// FindByName returns the domain matching the name.
	FindByName(context.Context, Queryer, string) (*models.DomainEntity, error)
}

// domainDao is the sqlite implementation of DomainDao.
type domainDao struct{}

// NewDomainDao creates a new DomainDao.
func NewDomainDao() DomainDao {
	return domainDao{}
}

func (domainDao) Insert(ctx context.Context, q Queryer, domain *models.DomainEntity) error {
	const query = `
		insert into "domains" ( "name" )
		values ( :name ) ;
	`

	result, err := execNamed(ctx, q, query, domain)
	if err != nil {
		return err
	}

	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	domain.ID, err = result.LastInsertId()
	return err
}

func (domainDao) Update(ctx context.Context, q Queryer, domain *models.DomainEntity) error {
	const query = `
		update "domains"
		set "name" = :name
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, domain)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (domainDao) Delete(ctx context.Context, q Queryer, domain *models.DomainEntity) error {
	const query = `
		delete from "domains"
		where "id" = :id ;
	`

	result, err := execNamed(ctx, q, query, domain)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

func (domainDao) FindAll(ctx context.Context, q Queryer) ([]models.DomainEntity, error) {
	const query = `
		select *
		from "domains"
		order by "name" asc ;
	`

	var domainSlice []models.DomainEntity

	if err := selectSlice(ctx, q, &domainSlice, query); err != nil {
		return nil, err
	}

	return domainSlice, nil
}

func (domainDao) FindByName(ctx context.Context, q Queryer, name string) (*models.DomainEntity, error) {
	const query = `
		select *
		from "domains"
		where "name" = $1
		limit 1;
	`

	var domain models.DomainEntity

	if err := selectOne(ctx, q, &domain, query, name); err != nil {
		return nil, err
	}

	return &domain, nil
}
