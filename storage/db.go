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
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/model"
)

func init() {
	viper.SetDefault("storage.database.filename", "data/db.sqlite")
}

type DB struct {
	conn *sql.DB
}

func NewDB() (*DB, error) {
	db, err := sql.Open("sqlite3", viper.GetString("storage.database.filename"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(
		`
		create table if not exists "mailboxes" (
			"id"       integer         primary key autoincrement ,
			"name"     varchar ( 64 )  unique ,
			"hash"     blob            not null
		) ;

		create table if not exists "mails" (
			"uuid"     char ( 36 )     primary key ,
			"date"     integer         not null ,
			"from"     varchar ( 256 ) not null ,
			"size"     integer         not null ,
			"offset"   integer         not null
		) ;

		create table if not exists "entries" (
			"mailbox"  integer         not null ,
			"mail"     char ( 36 )     not null ,

			primary key ( "mailbox", "mail" ) ,
			foreign key ( "mailbox" ) references "mailboxes" ( "id"   ) ,
			foreign key ( "mail"    ) references "mails"     ( "uuid" )
		) ;

		create table if not exists "queue" (
			"mail"     char ( 36 )     primary key ,
			"date"     integer         not null ,
			"attempts" integer         not null ,
			"to"       blob            not null ,

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
		tx.Rollback() // nolint:errcheck
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
		result, err := tx.Exec(
			`
			update "mailboxes"
			set "hash" = ?
			where "name" = ? ;
			`, hash, name)

		if err != nil {
			return err
		}

		ar, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if ar < 1 {
			return sql.ErrNoRows
		}

		return nil
	})
}

func hashPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(hash), err
}

func (d *DB) Authenticate(name, pass string) (int64, bool, error) {
	var (
		id int64
		ok bool
	)

	return id, ok, d.do(func(tx *sql.Tx) error {
		var hash []byte

		err := tx.QueryRow(
			`
			select "id", "hash"
			from "mailboxes"
			where "name" = ? ;
			`, name).Scan(&id, &hash)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}

			return err
		}

		ok = bcrypt.CompareHashAndPassword(hash, []byte(pass)) == nil
		return nil
	})
}

type Mail struct {
	ID     model.ID
	Date   time.Time
	From   *model.Address
	Size   int64
	Offset int64
}

func (d *DB) Mail(id model.ID) (*Mail, error) {
	var m Mail

	return &m, d.do(func(tx *sql.Tx) error {
		var _date int64

		err := tx.QueryRow(
			`
			select "date", "from", "size", "offset"
			from "mails"
			where "uuid" = ? ;
			`, id).Scan(&_date, &m.From, &m.Size, &m.Offset)

		if err != nil {
			return err
		}

		m.Date = time.Unix(_date, 0)
		return nil
	})
}

func (d *DB) AddMail(id model.ID, size, offset int64, envelope *model.Envelope) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			insert into "mails"
			( "uuid", "date", "from", "size", "offset" )
			values
			( ?, ?, ?, ?, ? ) ;
			`, id, envelope.Date.Unix(), envelope.From.String(), size, offset)

		return err
	})
}

type Entry struct {
	MailID model.ID
	Size   int64
}

func (d *DB) Entries(mailbox int64) ([]Entry, int64, error) {
	var (
		list  []Entry
		total int64
	)

	return list, total, d.do(func(tx *sql.Tx) error {
		rows, err := tx.Query(
			`
			select "m"."uuid", "m"."size"
			from "mails" as "m"
				inner join "entries" as "e"
					on "m"."uuid" = "e"."mail"
			where "e"."mailbox" = ?
			order by "m"."date" desc
			limit 1000 ;
			`, mailbox)

		if err != nil {
			return err
		}

		defer rows.Close() // nolint:errcheck

		var entry Entry

		for rows.Next() {
			if err := rows.Scan(&entry.MailID, &entry.Size); err != nil {
				return err
			}

			total += entry.Size
			list = append(list, entry)
		}

		return nil
	})
}

func (d *DB) AddEntries(mail model.ID, mailboxes []int64) error {
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

		defer stmt.Close() // nolint:errcheck

		for _, mailbox := range mailboxes {
			if _, err := stmt.Exec(mailbox, mail); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *DB) DeleteEntries(mails []model.ID, mailbox int64) error {
	return d.do(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(
			`
			delete from "entries"
			where "mailbox" = ?
			  and "mail" = ? ;
			`)

		if err != nil {
			return err
		}

		defer stmt.Close() // nolint:errcheck

		for _, mail := range mails {
			if _, err := stmt.Exec(mailbox, mail); err != nil {
				return err
			}
		}

		return err
	})
}

type QueueElement struct {
	MailID   model.ID
	Date     time.Time
	Attempts int
	To       []*model.Address
}

func (d *DB) PeekQueue() (*QueueElement, error) {
	var (
		element QueueElement
		_date   int64
		_to     []byte
	)

	return &element, d.do(func(tx *sql.Tx) error {
		err := tx.QueryRow(
			`
			select "mail", "date", "attempts", "to"
			from "queue"
			order by "date" asc
			limit 1 ;
			`).Scan(&element.MailID, &_date, &element.Attempts, &_to)

		if err != nil {
			return err
		}

		element.Date = time.Unix(_date, 0)
		return json.Unmarshal(_to, &element.To)
	})
}

func (d *DB) AddToQueue(mail model.ID, to []*model.Address) error {
	_to, err := json.Marshal(to)
	if err != nil {
		return err
	}

	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			insert into "queue"
			( "mail", "date", "attempts", "to" )
			values
			( ?, '0', '0', ? )
			`, mail, _to)

		return err
	})
}

func (d *DB) UpdateQueue(mail model.ID, to []*model.Address, date time.Time) error {
	_to, err := json.Marshal(to)
	if err != nil {
		return err
	}

	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			update "queue"
			set "date" = ? ,
			    "attempts" = "attempts" + 1 ,
			    "to" = ?
			where "mail" = ? ;
			`, date.Unix(), _to, mail)

		return err
	})
}

func (d *DB) DeleteFromQueue(mail model.ID) error {
	return d.do(func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`
			delete from "queue"
			where "mail" = ? ;
			`, mail)

		return err
	})
}

func (d *DB) DeleteOrphans() ([]model.ID, error) {
	var orphans []model.ID

	return orphans, d.do(func(tx *sql.Tx) error {
		rows, err := tx.Query(
			`
			select "uuid"
			from "mails"
			where (
					select count(*)
					from "entries"
					where "mail" = "uuid"
				  ) = 0
			  and (
			  		select count(*)
			  		from "queue"
			  		where "mail" = "uuid"
			  	  ) = 0 ;
			`)

		if err != nil {
			return err
		}

		defer rows.Close()

		var orphan model.ID

		for rows.Next() {
			if err := rows.Scan(&orphan); err != nil {
				return err
			}

			orphans = append(orphans, orphan)
		}

		stmt, err := tx.Prepare(
			`
			delete from "mails"
			where "uuid" = ? ;
			`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		for _, orphan := range orphans {
			if _, err := stmt.Exec(orphan); err != nil {
				return err
			}
		}

		return nil
	})
}
