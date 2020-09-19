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
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"sort"
	"time"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

// SendResult indicates the status of delivery for a collection of recipients.
type SendResult int

const (
	_ SendResult = 1 << iota
	// SomePending means at least one recipient is still pending, because of a transient error.
	SomePending
	// SomeFailed means at least one recipient permanently failed.
	SomeFailed
	// SomeSuccess means at least one recipient was delivered succesfully.
	SomeSuccess
)

func (s *SendResult) update(result SendResult) {
	*s |= result
}

func (s SendResult) check(result SendResult) bool {
	return s&result == result
}

const (
	smtpPort = "25"
)

// Courier handles the delivery of outbound mails.
type Courier struct {
	database *storage.Database
	blobs    *storage.Blobs
	hostname string
}

// NewCourier creates a new courier for delivery.
func NewCourier(database *storage.Database, blobs *storage.Blobs) *Courier {
	return &Courier{
		database: database,
		blobs:    blobs,
		hostname: viper.GetString("general.hostname"),
	}
}

// SendMail attempts to send a mail to all pending recipients. An error is only returned on database
// errors. Other errors are logged but only affect the SendResult. After the attempt, the mail
// attempt count and the status of all pending recipients is updated and stored in the database.
func (c *Courier) SendMail(ctx context.Context, mail *storage.Mail) (SendResult, error) {
	log.InfoContext(ctx).
		Str("mail", mail.ID).
		Int("attempts", mail.Attempts).
		Msg("attempting to send to pending recipients")

	var result SendResult

	recipients, err := c.findRecipients(ctx, mail)
	if err != nil {
		return result, err
	}

	for _, recipients := range recipients {
		domainResult := c.sendMailToDomain(ctx, mail, recipients)
		result.update(domainResult)
	}

	log.InfoContext(ctx).
		Str("mail", mail.ID).
		Int("result", int(result)).
		Msg("attempt completed")

	return result, c.saveAttempt(ctx, mail, recipients)
}

// saveAttempt updates the mail attempt count and writes it, as well as the status of all recipients
// to the database.
func (c *Courier) saveAttempt(ctx context.Context, mail *storage.Mail, recipients []recipientSlice) error {
	tx, err := c.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	mail.Attempts++
	mail.LastAttemptedAt.Int64 = time.Now().Unix()
	mail.LastAttemptedAt.Valid = true

	if err := queries.UpdateMail(tx, mail); err != nil {
		return err
	}

	for _, recipients := range recipients {
		for _, recipient := range recipients {
			if err := queries.UpdateRecipient(tx, &recipient); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// sendMailToDomain attempts to send a mail to all recipients of the same domain. This method
// expects all recipients to have the same domain and therefore uses the domain of the first element
// for the delivery to all recipients. This method does not return an error.
func (c *Courier) sendMailToDomain(ctx context.Context, mail *storage.Mail, recipients recipientSlice) SendResult {
	domain := recipients[0].ForwardPath.Domain()

	log.DebugContext(ctx).
		Str("mail", mail.ID).
		Str("domain", domain).
		Msg("sending mail to recipients of domain")

	// net.LookupMX already returns a sorted slice
	records, err := net.LookupMX(domain)
	if err != nil {
		return SomePending
	}

	for _, record := range records {
		log.DebugContext(ctx).
			Str("mail", mail.ID).
			Str("domain", domain).
			Str("host", record.Host).
			Msg("trying mx host")

		if err := c.sendMailToHost(mail, recipients, record.Host); err != nil {
			log.DebugContext(ctx).
				Str("mail", mail.ID).
				Str("domain", domain).
				Str("host", record.Host).
				Err(err).
				Msg("error during host attempt")

			if isPermanentErr(err) {
				// if delivery failed due to a "permanent" smtp error, the delivery to all
				// recipients fails.

				log.InfoContext(ctx).
					Str("mail", mail.ID).
					Str("domain", domain).
					Str("host", record.Host).
					Err(err).
					Msg("delivery failed for all recipients permanently")

				recipients.updateAllStatus(storage.StatusFailed)
				return SomeFailed
			}

			if !isTransientErr(err) {
				// if the error is not a "permanent" or "transient" smtp error, it is something
				// totally different. In that case we try the next mx record.

				continue
			}
		}

		// if there was no error or the error was a "transient" smtp error, we will not attempt any
		// more mx records and just return what we have.
		break
	}

	return recipients.countResult()
}

// sendMailToHost attempts to send a mail to an actual host. The host is the result of a previous
// mx record lookup for the domain of the recipients.
func (c *Courier) sendMailToHost(mail *storage.Mail, recipients recipientSlice, host string) error {
	client, err := smtp.Dial(net.JoinHostPort(host, smtpPort))
	if err != nil {
		return err
	}

	defer client.Close()

	if err := c.initClient(client, host); err != nil {
		return err
	}

	if err := c.copyEnvelope(client, mail, recipients); err != nil {
		return err
	}

	if err := c.copyData(client, mail); err != nil {
		return err
	}

	if client.Quit(); err != nil {
		return err
	}

	recipients.updatePendingStatus(storage.StatusDelivered)
	return nil
}

// initClient says hello to the server and upgrades to tls, if available.
func (c *Courier) initClient(client *smtp.Client, host string) error {
	if err := client.Hello(c.hostname); err != nil {
		return err
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		config := tls.Config{
			ServerName: host,
		}

		return client.StartTLS(&config)
	}

	return nil
}

// copyEnvelope sends the return- and forward-paths of the mail.
func (c *Courier) copyEnvelope(client *smtp.Client, mail *storage.Mail, recipients recipientSlice) error {
	if err := client.Mail(mail.ReturnPath); err != nil {
		return err
	}

	for i, recipient := range recipients {
		if err := client.Rcpt(recipient.ForwardPath.String()); err != nil {
			switch {
			case isPermanentErr(err):
				recipients[i].Status = storage.StatusFailed

			case isTransientErr(err):
				// stay pending

			default:
				return err
			}
		}
	}

	return nil
}

// copyData writes the mail content.
func (c *Courier) copyData(client *smtp.Client, mail *storage.Mail) error {
	w, err := client.Data()
	if err != nil {
		return err
	}

	r, err := c.blobs.Reader(mail.ID)
	if err != nil {
		return err
	}

	defer r.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return w.Close()
}

// isPermanentErr tests if an error is an smtp error and if it has a 5xx code.
func isPermanentErr(err error) bool {
	var protoError *textproto.Error
	if errors.As(err, &protoError) {
		return protoError.Code >= 500 && protoError.Code < 600
	}

	return false
}

// isPermanentErr tests if an error is an smtp error and if it has a 4xx code.
func isTransientErr(err error) bool {
	var protoError *textproto.Error
	if errors.As(err, &protoError) {
		return protoError.Code >= 400 && protoError.Code < 500
	}

	return false
}

// recipientSlice is a sortable slice of recipients.
// The order is defined by their forward path domains.
type recipientSlice []storage.Recipient

func (r recipientSlice) Len() int {
	return len(r)
}

func (r recipientSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r recipientSlice) Less(i, j int) bool {
	return r[i].ForwardPath.Domain() < r[j].ForwardPath.Domain()
}

func (r recipientSlice) updateAllStatus(status storage.DeliveryStatus) {
	for i := 0; i < len(r); i++ {
		r[i].Status = status
	}
}

func (r recipientSlice) updatePendingStatus(status storage.DeliveryStatus) {
	for i, recipient := range r {
		if recipient.Status == storage.StatusPending {
			r[i].Status = status
		}
	}
}

func (r recipientSlice) groupByDomain() []recipientSlice {
	if len(r) == 0 {
		return nil
	}

	if len(r) == 1 {
		return []recipientSlice{r}
	}

	// sort recipients by domain
	sort.Sort(r)

	var (
		groups []recipientSlice
		left   int
		domain = r[0].ForwardPath.Domain()
	)

	// group recipients into subslices by domain
	for right := 1; right < len(r); right++ {
		recipientDomain := r[right].ForwardPath.Domain()

		if domain != recipientDomain {
			groups = append(groups, r[left:right])
			domain = recipientDomain
			left = right
		}
	}

	return append(groups, r[left:])
}

func (r recipientSlice) countResult() SendResult {
	var result SendResult

	for _, recipient := range r {
		switch recipient.Status {
		case storage.StatusPending:
			result.update(SomePending)

		case storage.StatusFailed:
			result.update(SomeFailed)

		case storage.StatusDelivered:
			result.update(SomeSuccess)
		}
	}

	return result
}

func (c *Courier) findRecipients(ctx context.Context, mail *storage.Mail) ([]recipientSlice, error) {
	tx, err := c.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	recipients, err := queries.FindPendingRecipients(tx, mail)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return recipientSlice(recipients).groupByDomain(), nil
}
