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

	"github.com/lukasdietrich/briefmail/model"
)

type EntryKind int

const (
	Local EntryKind = iota
	Forward
)

type Entry struct {
	Kind    EntryKind
	Mailbox *int64
	Forward *model.Address
}

func (e *Entry) String() string {
	switch e.Kind {
	case Local:
		return fmt.Sprintf("local(mailbox=%d)", *e.Mailbox)
	case Forward:
		return fmt.Sprintf("forward(to=%s)", e.Forward)
	}

	return ""
}

type Book struct {
	entries map[string]map[string]*Entry
}

func (b *Book) Lookup(addr *model.Address) (*Entry, bool) {
	entry, ok := lookupInDomain(b.entries[addr.Domain], addr)
	if ok {
		return entry, true
	}

	entry, ok = lookupInDomain(b.entries["*"], addr)
	return entry, ok
}

func lookupInDomain(domain map[string]*Entry, addr *model.Address) (*Entry, bool) {
	if domain == nil {
		return nil, false
	}

	entry, ok := domain[addr.User]
	if ok {
		return entry, true
	}

	entry, ok = domain["*"]
	return entry, ok
}
