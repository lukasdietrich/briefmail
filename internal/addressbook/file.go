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

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/model"
	"github.com/lukasdietrich/briefmail/internal/normalize"
	"github.com/lukasdietrich/briefmail/internal/storage"
)

func init() {
	viper.SetDefault("general.domains", []string{"localhost"})
	viper.SetDefault("addressbook.filename", "_example/addressbook.toml")
}

// [mailboxes]
//   "name" = [ "address1@domain1", "address2@domain1" ]

type fileFormat struct {
	Mailboxes map[string][]string
}

func makeDomainSet() (*normalize.Set, error) {
	return normalize.NewSet(viper.GetStringSlice("general.domains"), normalize.Domain)
}

func Parse(db *storage.DB) (Addressbook, error) {
	domains, err := makeDomainSet()
	if err != nil {
		return nil, err
	}

	var (
		fileName    = viper.GetString("addressbook.filename")
		data        fileFormat
		addressbook addressbook
	)

	if _, err := toml.DecodeFile(fileName, &data); err != nil {
		return nil, err
	}

	addressbook.entries = make(map[string]map[string]*Entry)
	addressbook.domains = domains

	for name, addresses := range data.Mailboxes {
		mailbox, err := db.Mailbox(name)
		if err != nil {
			return nil, fmt.Errorf("addressbook: unknown mailbox '%s'", name)
		}

		for _, address := range addresses {
			addr, err := model.ParseAddress(address)
			if err != nil {
				return nil, err
			}

			if _, ok := addressbook.entries[addr.Domain]; !ok {
				addressbook.entries[addr.Domain] = make(map[string]*Entry)
			}

			addressbook.entries[addr.Domain][addr.User] = &Entry{
				Kind:    Local,
				Mailbox: &mailbox,
			}
		}
	}

	logrus.Debug("addressbook:")
	for domain, entries := range addressbook.entries {
		logrus.Debugf("- domain: \"%s\"", domain)

		for user, entry := range entries {
			logrus.Debugf("  - user: \"%s\" => %s", user, entry)
		}
	}

	return &addressbook, nil
}
