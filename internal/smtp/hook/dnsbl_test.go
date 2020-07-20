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

package hook

import (
	"net"
	"testing"

	"github.com/lukasdietrich/briefmail/internal/mails"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatReverseIP(t *testing.T) {
	for ip, expected := range map[string]string{
		"192.0.2.99":                "99.2.0.192.",
		"111.122.133.144":           "144.133.122.111.",
		"2001:db8:1:2:3:4:567:89ab": "b.a.9.8.7.6.5.0.4.0.0.0.3.0.0.0.2.0.0.0.1.0.0.0.8.b.d.0.1.0.0.2.",
	} {
		t.Run(ip, func(t *testing.T) {
			actual := formatReverseIP(net.ParseIP(ip))
			assert.Equal(t, expected, actual)
		})
	}
}

func TestDnsblHook(t *testing.T) {
	const (
		badIP  = "127.0.0.2"
		goodIP = "127.0.0.1"
	)

	hook := makeDnsblHook()
	require.NotNil(t, hook)

	t.Run("BadRecord", func(t *testing.T) {
		result, err := hook(false, net.ParseIP(badIP), mails.ZeroAddress)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Reject)
		assert.Equal(t, 550, result.Code)
	})

	t.Run("GoodRecord", func(t *testing.T) {
		result, err := hook(false, net.ParseIP(goodIP), mails.ZeroAddress)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Reject)
		assert.Equal(t, 0, result.Code)
	})

	t.Run("Submission", func(t *testing.T) {
		result, err := hook(true, net.ParseIP(badIP), mails.ZeroAddress)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Reject)
		assert.Equal(t, 0, result.Code)
	})
}
