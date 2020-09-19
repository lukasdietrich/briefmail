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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/storage"
	"github.com/lukasdietrich/briefmail/internal/storage/queries"
)

var (
	errCouldNotConnect = errors.New("could not connect to any mx host")
)

func init() {
	viper.SetDefault("mail.queue.delays", []int{20, 20, 20, 60, 180, 600})
	viper.SetDefault("mail.queue.giveUpAfter", 4320) // 3 days
}

// Queue coordinates the delivery attempts of outbound mails.
type Queue struct {
	database *storage.Database
	courier  *Courier

	delaysInSeconds    []int64
	giveUpAfterSeconds int64

	mu    sync.Mutex  // protect the timer and busy flag.
	timer *time.Timer // timer is used to wait for the next delivery attempt.
	busy  bool        // busy is a flag to indicate if a delivery attempt is currently in progress.
}

// NewQueue creates a new Queue.
func NewQueue(database *storage.Database, courier *Courier) (*Queue, error) {
	var (
		delaysInMinutes    = viper.GetIntSlice("mail.queue.delays")
		giveUpAfterMinutes = viper.GetInt("mail.queue.giveUpAfter")

		delaysInSeconds    = make([]int64, len(delaysInMinutes))
		giveUpAfterSeconds = int64(giveUpAfterMinutes) * 60
	)

	if len(delaysInMinutes) == 0 {
		return nil, fmt.Errorf("mail.queue.delays may not be empty")
	}

	if giveUpAfterMinutes <= 0 {
		return nil, fmt.Errorf("mail.queue.giveUpAfter must be positive")
	}

	for i, delayInMinutes := range delaysInMinutes {
		delaysInSeconds[i] = int64(delayInMinutes) * 60
	}

	queue := Queue{
		database: database,
		courier:  courier,

		delaysInSeconds:    delaysInSeconds,
		giveUpAfterSeconds: giveUpAfterSeconds,
	}

	queue.WakeUp(context.Background())
	return &queue, nil
}

// WakeUp schedules the next pending mail for delivery.
// Only one delivery will be executed at a time.
func (q *Queue) WakeUp(ctx context.Context) {
	defer q.mu.Unlock()
	q.mu.Lock()

	// When a delivery attempt is currently in progress, do not schedule another. When the attempt
	// is done, the next attempt will be scheduled.
	if !q.busy {

		// If a timer is already set, override it.
		if q.timer != nil {
			q.timer.Stop()
			q.timer = nil
		}

		// schedule the next attempt.
		if err := q.schedule(ctx); err != nil {
			log.ErrorContext(ctx).
				Err(err).
				Msg("could not schedule outbound delivery attempt")
		}
	}
}

// schedule finds the next pending mail. If any is present, an attempt is scheduled.
func (q *Queue) schedule(ctx context.Context) error {
	tx, err := q.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	mail, err := queries.FindNextPendingMail(tx)
	if err != nil {
		// if no pending mail exists, do not schedule an attempt.
		if storage.IsErrNoRows(err) {
			log.DebugContext(ctx).Msg("no mail is pending")
			return nil
		}

		return err
	}

	attemptAt := q.determineNextAttemptTime(mail)
	waitDuration := time.Until(attemptAt)

	if waitDuration < 0 {
		// if the next attempt should have already happend, schedule it now.
		waitDuration = 0
	}

	log.InfoContext(ctx).
		Str("mail", mail.ID).
		Int("attempts", mail.Attempts).
		Dur("waitDuration", waitDuration).
		Msg("scheduling delivery attempt")

	// execute attemptMail using the timer, even if waitDuration is 0, because we are currently
	// holding a lock. If we call it directly, we have a deadlock.
	q.timer = time.AfterFunc(waitDuration, q.attemptDelivery(mail))

	return tx.Commit()
}

// determineNextAttemptTime calculates the time for the next delivery attempt of a mail. The next
// attempt is at `LastAttemptTime + delay(attempt)`. The first attempt has no delay.
func (q *Queue) determineNextAttemptTime(mail *storage.Mail) time.Time {
	scheduleTime := mail.ReceivedAt

	if mail.LastAttemptedAt.Valid {
		scheduleTime = mail.LastAttemptedAt.Int64

		if mail.Attempts < len(q.delaysInSeconds) {
			scheduleTime += q.delaysInSeconds[mail.Attempts]
		} else {
			scheduleTime += q.delaysInSeconds[len(q.delaysInSeconds)-1]
		}
	}

	return time.Unix(scheduleTime, 0)
}

func (q *Queue) attemptDelivery(mail *storage.Mail) func() {
	ctx := context.TODO()
	ctx = log.WithOrigin(ctx, "queue")

	return func() {
		if q.setBusy(true) {
			log.WarnContext(ctx).
				Str("mail", mail.ID).
				Msg("another attempt is already in progress")

			return
		}

		result, err := q.courier.SendMail(ctx, mail)
		if err != nil {
			log.ErrorContext(ctx).
				Err(err).
				Msg("could not attempt delivery")
		} else {
			q.handleDeliveryResult(ctx, mail, result)
		}

		q.setBusy(false)
		q.WakeUp(ctx)
	}
}

func (q *Queue) handleDeliveryResult(ctx context.Context, mail *storage.Mail, result SendResult) {
	var (
		hasPending = result.check(SomePending)
		hasFailed  = result.check(SomeFailed)
	)

	if hasPending {
		log.InfoContext(ctx).
			Str("mail", mail.ID).
			Msg("there are still pending recipients")
	}

	if hasFailed {
		log.WarnContext(ctx).
			Str("mail", mail.ID).
			Msg("some recipients failed. notification mail is not yet implemented")
	}
}

func (q *Queue) setBusy(busy bool) (wasBusy bool) {
	defer q.mu.Unlock()
	q.mu.Lock()

	wasBusy, q.busy = q.busy, busy
	return
}
