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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukasdietrich/briefmail/internal/models"
)

func TestRecipientGroupByDomainEmpty(t *testing.T) {
	var recipients recipientSlice
	groups := recipients.groupByDomain()

	assert.Equal(t, 0, len(groups))
}

func TestRecipientGroupByDomainOne(t *testing.T) {
	recipients := recipientSlice{
		fakeRecipient(t, "someone@example.com"),
	}

	actual := recipients.groupByDomain()
	expected := []recipientSlice{{
		fakeRecipient(t, "someone@example.com"),
	}}

	assert.Equal(t, expected, actual)
}

func TestRecipientGroupByDomainMany(t *testing.T) {
	recipients := recipientSlice{
		fakeRecipient(t, "user-1@domain-2"),
		fakeRecipient(t, "user-2@domain-1"),
		fakeRecipient(t, "user-3@domain-2"),
		fakeRecipient(t, "user-4@domain-3"),
		fakeRecipient(t, "user-5@domain-3"),
	}

	actual := recipients.groupByDomain()
	expected := []recipientSlice{
		{
			fakeRecipient(t, "user-2@domain-1"),
		},
		{
			fakeRecipient(t, "user-1@domain-2"),
			fakeRecipient(t, "user-3@domain-2"),
		},
		{
			fakeRecipient(t, "user-4@domain-3"),
			fakeRecipient(t, "user-5@domain-3"),
		},
	}

	assert.Equal(t, expected, actual)
}

func fakeRecipient(t *testing.T, forwardPath string) models.RecipientEntity {
	address, err := models.Parse(forwardPath)
	require.NoError(t, err)

	return models.RecipientEntity{
		ForwardPath: address,
	}
}
