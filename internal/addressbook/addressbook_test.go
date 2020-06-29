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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lukasdietrich/briefmail/internal/model"
	"github.com/lukasdietrich/briefmail/internal/normalize"
)

func TestSimple(t *testing.T) {
	var (
		user1AtHost1 = makeEntry(0)
		user2AtHost1 = makeEntry(1)
		user1AtHost2 = makeEntry(2)
		user2AtHost2 = makeEntry(3)
	)

	domains, err := normalize.NewSet([]string{"host1", "host2"}, normalize.Domain)
	assert.Nil(t, err)

	addressbook := addressbook{
		domains: domains,
		entries: map[string]map[string]*Entry{
			"host1": {
				"user1": user1AtHost1,
				"user2": user2AtHost1,
			},
			"host2": {
				"user1": user1AtHost2,
				"user2": user2AtHost2,
			},
		},
	}

	for addr, entry := range map[string]*Entry{
		"user1@host1": user1AtHost1,
		"user1@host2": user1AtHost2,
		"user2@host1": user2AtHost1,
		"user2@host2": user2AtHost2,
		"user3@host1": nil,
		"user1@host3": {
			Kind:    Remote,
			Address: mustAddress("user1@host3"),
		},
	} {
		t.Run(addr, func(t *testing.T) {
			actual := addressbook.Lookup(mustAddress(addr))
			assert.Equal(t, entry, actual)
		})
	}
}

func TestWildcard(t *testing.T) {
	var (
		user1AtHost1 = makeEntry(0)
		anyAtHost1   = makeEntry(1)
		user1AtAny   = makeEntry(2)
		anyAtAny     = makeEntry(3)
	)

	domains, err := normalize.NewSet([]string{"host1", "host2"}, normalize.Domain)
	assert.Nil(t, err)

	addressbook := addressbook{
		domains: domains,
		entries: map[string]map[string]*Entry{
			"host1": {
				"user1": user1AtHost1,
				"*":     anyAtHost1,
			},
			"*": {
				"user1": user1AtAny,
				"*":     anyAtAny,
			},
		},
	}

	for addr, entry := range map[string]*Entry{
		"user1@host1": user1AtHost1,
		"user2@host1": anyAtHost1,
		"user1@host2": user1AtAny,
		"user2@host2": anyAtAny,
	} {
		t.Run(addr, func(t *testing.T) {
			actual := addressbook.Lookup(mustAddress(addr))
			assert.Equal(t, entry, actual)
		})
	}
}

func mustAddress(raw string) *model.Address {
	addr, err := model.ParseAddress(raw)
	if err != nil {
		panic(err)
	}

	return addr
}

func makeEntry(id int64) *Entry {
	return &Entry{Mailbox: &id}
}
