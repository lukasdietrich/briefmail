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
	"github.com/lukasdietrich/argon2go"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/storage"
)

// ErrPasswordMismatch is returned when a password does not match the hash.
var ErrPasswordMismatch = argon2go.ErrMismatch

func init() {
	viper.SetDefault("crypto.argon2.hashlength", 32)
	viper.SetDefault("crypto.argon2.saltlength", 16)
	viper.SetDefault("crypto.argon2.time", 2)
	viper.SetDefault("crypto.argon2.memory", 64*1024)
	viper.SetDefault("crypto.argon2.threads", 4)
}

// Hash applies the argon2id hashing algorithm to a password and stores the hash in the mailbox.
// The options used for hashing are determined using viper.
func Hash(mailbox *storage.Mailbox, pass []byte) (err error) {
	opts := argon2go.Options{
		Time:       viper.GetUint32("crypto.argon2.time"),
		Memory:     viper.GetUint32("crypto.argon2.memory"),
		Threads:    uint8(viper.GetUint32("crypto.argon2.threads")),
		HashLength: viper.GetUint32("crypto.argon2.hashlength"),
		SaltLength: viper.GetUint32("crypto.argon2.saltlength"),
	}

	mailbox.Hash, err = argon2go.Hash(pass, &opts)
	return
}

// Verify checks if a password matches the mailboxes hash. If the password does not match
// ErrPasswordMismatch is returned. There may occur other, technical errors.
func Verify(mailbox *storage.Mailbox, pass []byte) error {
	return argon2go.Verify(pass, mailbox.Hash)
}
