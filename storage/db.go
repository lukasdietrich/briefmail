// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
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

package storage

import (
	"database/sql"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lukasdietrich/briefmail/model"
)

type DB struct {
	conn *sql.DB
}

func NewDB(fileName string) (*DB, error) {
	db, err := sql.Open("sqlite3", fileName)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(
		`
		create table if not exists "mailboxes" (
			"id"      integer         primary key autoincrement ,
			"name"    varchar ( 64 )  unique ,
			"hash"    blob            not null
		) ;

		create table if not exists "mails" (
			"uuid"    char ( 36 )     primary key ,
			"date"    integer         not null ,
			"from"    varchar ( 256 ) not null ,
			"size"    integer         not null
		) ;

		create table if not exists "entries" (
			"mailbox" integer         not null ,
			"mail"    char ( 36 )     not null ,

			primary key ( "mailbox", "mail" ) ,
			foreign key ( "mailbox" ) references "mailboxes" ( "id"   ) ,
			foreign key ( "mail"    ) references "mails"     ( "uuid" )
		) ;

		create table if not exists "queue" (
			"id"      integer         primary key autoincrement ,
			"mail"    char ( 36 )     not null ,
			"to"      varchar ( 256 ) not null ,
			"count"   integer         not null ,
			"date"    integer         not null ,

			foreign key ( "mail" ) references "mails" ( "uuid" )
		) ;
		`)

	return &DB{conn: db}, err
}

func (d *DB) do(fn func(*sql.Tx) error) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DB) Mailbox(name string) (int64, error) {
	var id int64

	return id, d.do(func(tx *sql.Tx) error {
		return tx.QueryRow(
			`
			select "id"
			from "mailboxes"
			where "name" = ? ;
			`, name).Scan(&id)
	})
}

func (d *DB) AddMailbox(name, pass string) (int64, error) {
	hash, err := hashPassword(pass)
	if err != nil {
		return -1, err
	}

	var id int64

	return id, d.do(func(tx *sql.Tx) error {
		result, err := tx.Exec(
			`
			insert into "mailboxes"
			( "name", "hash" )
			values
			( ?, ? ) ;
			`, name, hash)

		if err != nil {
			return err
		}

		id, err = result.LastInsertId()
		return err
	})
}

func (d *DB) SetPassword(name, pass string) error {
	hash, err := hashPassword(pass)
	if err != nil {
		return err
	}

	return d.do(func(tx *sql.Tx) error {
		_, err = tx.Exec(
			`
			update "mailboxes"
			set "hash" = ?
			where "name" = ? ;
			`, hash, name)

		return err
	})
}

func hashPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(hash), err
}

func (d *DB) AddMail(id uuid.UUID, size int64, envelope *model.Envelope) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			insert into "mails"
			( "uuid", "date", "from", "size" )
			values
			( ?, ?, ?, ? ) ;
			`, id, envelope.Date.Unix(), envelope.From.String(), size)

		return err
	})
}

func (d *DB) DeleteMail(id uuid.UUID) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			delete from "mails"
			where "uuid" = ? ;
			`, id)

		return err
	})
}

func (d *DB) AddEntries(mail uuid.UUID, mailboxes []int64) error {
	return d.do(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(
			`
			insert into "entries"
			( "mailbox", "mail" )
			values
			( ?, ? ) ;
			`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		for _, mailbox := range mailboxes {
			if _, err := stmt.Exec(mailbox, mail); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *DB) DeleteEntry(mail uuid.UUID, mailbox int64) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			delete from "entries"
			where "mailbox" = ?
			  and "mail" = ? ;
			`, mailbox, mail)

		return err
	})
}

func (d *DB) AddToQueue(mail uuid.UUID, to []*model.Address) error {
	return d.do(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(
			`
			insert into "queue"
			( "mail", "to", "count", "date" )
			values
			( ?, ?, '0', '0' )
			`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		for _, t := range to {
			if _, err := stmt.Exec(mail, t.String()); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *DB) DeleteFromQueue(id int64) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			delete from "queue"
			where "id" = ? ;
			`, id)

		return err
	})
}

func (d *DB) IsOrphan(mail uuid.UUID) (bool, error) {
	var isOrphan bool

	return isOrphan, d.do(func(tx *sql.Tx) error {
		var count int

		err := tx.QueryRow(
			`
			select count(*)
			from "entries"
			where "mail" = ? ;
			`, mail).Scan(&count)

		if err != nil || count > 0 {
			return err
		}

		err = tx.QueryRow(
			`
			select count(*)
			from "queue"
			where "mail" = ? ;
			`, mail).Scan(&count)

		isOrphan = count <= 0
		return err
	})
}