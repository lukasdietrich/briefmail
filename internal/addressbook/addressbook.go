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

package addressbook

import (
	"fmt"

	"github.com/lukasdietrich/briefmail/internal/mails"
)

type EntryKind int

const (
	Local EntryKind = iota
	Forward
	Remote
)

type Entry struct {
	Kind EntryKind

	Mailbox *int64
	Address mails.Address
}

func (e *Entry) String() string {
	switch e.Kind {
	case Local:
		return fmt.Sprintf("local(mailbox=%d)", *e.Mailbox)
	case Forward:
		return fmt.Sprintf("forward(address=%s)", e.Address)
	case Remote:
		return fmt.Sprintf("remote(address=%s)", e.Address)
	}

	return ""
}

type Addressbook interface {
	Lookup(mails.Address) *Entry
}

type addressbook struct {
	domains *Set
	entries map[string]map[string]*Entry
}

func (b *addressbook) Lookup(addr mails.Address) *Entry {
	if !b.domains.Contains(addr.Domain()) {
		return &Entry{
			Kind:    Remote,
			Address: addr,
		}
	}

	if entry := lookupInDomain(b.entries[addr.Domain()], addr); entry != nil {
		return entry
	}

	return lookupInDomain(b.entries["*"], addr)
}

func lookupInDomain(domain map[string]*Entry, addr mails.Address) *Entry {
	if domain == nil {
		return nil
	}

	if entry := domain[addr.LocalPart()]; entry != nil {
		return entry
	}

	return domain["*"]
}
