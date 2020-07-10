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

package delivery

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/dns"
	"github.com/lukasdietrich/briefmail/internal/model"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

var (
	errCouldNotConnect = errors.New("could not connect to any mx host")
)

type QueueWorker struct {
	DB    *storage.DB
	Blobs *storage.Blobs

	lock  sync.Mutex  `wire:"-"`
	alarm *time.Timer `wire:"-"`
	busy  bool        `wire:"-"`
}

func (q *QueueWorker) WakeUp() {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.alarm != nil {
		q.alarm.Stop()
		q.alarm = nil
	}

	if !q.busy {
		q.busy = true
		go q.work()
	}
}

func (q *QueueWorker) sleep(d time.Duration) {
	q.alarm = time.AfterFunc(d, q.WakeUp)
}

func (q *QueueWorker) next() (*storage.QueueElement, time.Duration, error) {
	elem, err := q.DB.PeekQueue()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}

		return nil, 0, err
	}

	if sleep := elem.Date.Sub(time.Now()); sleep > 0 {
		return nil, sleep, nil
	}

	return elem, 0, nil
}

func (q *QueueWorker) work() {
	cleaner := storage.NewCleaner(q.DB, q.Blobs)

	for {
		q.lock.Lock()

		elem, sleep, err := q.next()
		if err != nil {
			log.Error(err)
			time.Sleep(time.Minute * 5)

			continue
		}

		if elem == nil {
			q.busy = false

			if sleep > 0 {
				q.sleep(sleep)
			}
		}

		q.lock.Unlock()

		if elem == nil {
			break
		}

		q.do(elem)
		cleaner.Clean()
	}
}

func (q *QueueWorker) do(elem *storage.QueueElement) {
	log := log.WithFields(logrus.Fields{
		"mail":    elem.MailID,
		"attempt": elem.Attempts,
	})

	log.Info("attempting outbound delivery")

	mail, err := q.DB.Mail(elem.MailID)
	if err != nil {
		log.Error(err)
		return
	}

	var (
		delivered     []*model.Address
		undeliverable []*model.Address
		pending       []*model.Address
	)

	for domain, addresses := range addressesByDomain(elem.To) {
		var c client

		if err := c.connect(domain); err != nil {
			pending = append(pending, addresses...)
			continue
		}

		defer c.close()

		r, err := q.Blobs.OffsetReader(mail.ID, mail.Offset)
		if err != nil {
			pending = append(pending, addresses...)
			continue
		}

		hostname := viper.GetString("general.hostname")
		err = c.send(r, hostname, mail.From, addresses)
		r.Close()

		if err != nil {
			pending = append(pending, addresses...)
			continue
		}

		delivered = append(delivered, c.delivered...)
		undeliverable = append(undeliverable, c.undeliverable...)
		pending = append(pending, c.pending...)
	}

	if len(pending) > 0 {
		tryAgain, nextAttempt := scheduleAttempt(elem.Attempts + 1)

		if tryAgain {
			err := q.DB.UpdateQueue(elem.MailID, pending, nextAttempt)
			if err != nil {
				log.Error(err)
			}
		} else {
			undeliverable = append(undeliverable, pending...)
			pending = nil
		}
	}

	if len(undeliverable) > 0 {
		log.WithField("to", undeliverable).
			Warn("could not deliver to some recipients")
	}

	if len(pending) == 0 && len(undeliverable) == 0 {
		log.Info("delivered mail to all recipients")

		if err := q.DB.DeleteFromQueue(elem.MailID); err != nil {
			log.Error(err)
		}
	}
}

func scheduleAttempt(attempt int) (bool, time.Time) {
	now := time.Now()

	switch true {
	case attempt < 5:
		return true, now.Add(time.Minute * 10)

	case attempt < 10:
		return true, now.Add(time.Minute * 30)

	case attempt < 30:
		return true, now.Add(time.Minute * 60)
	}

	return false, now
}

func addressesByDomain(addresses []*model.Address) map[string][]*model.Address {
	domains := make(map[string][]*model.Address)

	for _, addr := range addresses {
		domains[addr.Domain] = append(domains[addr.Domain], addr)
	}

	return domains
}

type client struct {
	conn   net.Conn
	client *smtp.Client

	delivered     []*model.Address
	undeliverable []*model.Address
	pending       []*model.Address
}

func (c *client) close() {
	if c.client != nil {
		c.client.Quit()
	}

	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *client) connect(domain string) error {
	records, err := dns.QueryMX(domain)
	if err != nil {
		return err
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Preference < records[j].Preference
	})

	for _, record := range records {
		c.conn, err = net.Dial("tcp", net.JoinHostPort(record.Mx, "25"))
		if err != nil {
			continue
		}

		c.client, err = smtp.NewClient(c.conn, record.Mx)
		if err != nil {
			c.conn.Close()
			continue
		}

		return nil
	}

	return errCouldNotConnect
}

func (c *client) send(r io.Reader, hostname string, from *model.Address, to []*model.Address) error {
	if err := c.client.Hello(hostname); err != nil {
		return err
	}

	if ok, _ := c.client.Extension("STARTTLS"); ok {
		err := c.client.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
	}

	if err := c.client.Mail(from.String()); err != nil {
		return err
	}

	for _, addr := range to {
		if err := c.client.Rcpt(addr.String()); err != nil {
			if _err, ok := err.(*textproto.Error); ok {
				if _err.Code == 550 {
					c.undeliverable = append(c.undeliverable, addr)
					continue
				}

				c.pending = append(c.pending, addr)
				continue
			}

			return err
		}

		c.delivered = append(c.delivered, addr)
	}

	w, err := c.client.Data()
	if err != nil {
		return err
	}

	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}
