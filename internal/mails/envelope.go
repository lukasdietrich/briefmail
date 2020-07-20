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

package mails

import (
	"net"
	"time"
)

// Envelope stores the information about an email before the actual content is
// read. It is basically what a real envelope is to mail.
type Envelope struct {
	// Helo is the string provided by an smtp client when greeting the server.
	Helo string
	// Addr is the remote address of the sender.
	Addr net.IP
	// Date is the time when the data transmission begins.
	Date time.Time
	// From is the email-address of te sender.
	From Address
	// To is a list of recipient email-addresses.
	To []Address
}
