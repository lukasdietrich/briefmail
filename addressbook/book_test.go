package addressbook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lukasdietrich/briefmail/model"
)

func TestSimple(t *testing.T) {
	var (
		user1AtHost1 = makeEntry(0)
		user2AtHost1 = makeEntry(1)
		user1AtHost2 = makeEntry(2)
		user2AtHost2 = makeEntry(3)
	)

	book := Book{
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
		"user1@host3": nil,
	} {
		t.Run(addr, func(t *testing.T) {
			actual, ok := book.Lookup(mustAddress(addr))
			assert.Equal(t, entry != nil, ok)
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

	book := Book{
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
			actual, ok := book.Lookup(mustAddress(addr))
			assert.True(t, ok)
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
