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

package crypto

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/lukasdietrich/briefmail/internal/models"
)

const testHash = "$argon2id$v=19$m=128,t=4,p=3$bNXFICAXMjc$UFUBBgeLPRfLZCekIoSEoQ"

func TestHash(t *testing.T) {
	viper.Set("security.crypto.argon2.hashlength", 16)
	viper.Set("security.crypto.argon2.saltlength", 8)
	viper.Set("security.crypto.argon2.time", 4)
	viper.Set("security.crypto.argon2.memory", 128)
	viper.Set("security.crypto.argon2.threads", 3)

	var credentials models.MailboxCredentialEntity

	assert.NoError(t, Hash(&credentials, []byte("hunter2")))
	assert.Len(t, credentials.Hash, len(testHash))
	assert.Contains(t, credentials.Hash, "$argon2id$v=19$m=128,t=4,p=3$")
}

func TestVerifySuccessful(t *testing.T) {
	credentials := models.MailboxCredentialEntity{Hash: testHash}
	assert.NoError(t, Verify(&credentials, []byte("hunter2")))
}

func TestVerifyWrongPassword(t *testing.T) {
	credentials := models.MailboxCredentialEntity{Hash: testHash}
	assert.Equal(t, ErrPasswordMismatch, Verify(&credentials, []byte("hunter3")))
}
