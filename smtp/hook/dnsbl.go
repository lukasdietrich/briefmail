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

package hook

import (
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/dns"
	"github.com/lukasdietrich/briefmail/model"
)

func makeDnsblHook() FromHook {
	server := viper.GetString("hook.dnsbl.server")
	logrus.Debugf("hook: registering dnsbl hook (server=%s)", server)

	return func(submission bool, ip net.IP, _ *model.Address) (*Result, error) {
		if submission || ip.To4() == nil {
			return &Result{}, nil
		}

		log := logrus.WithFields(logrus.Fields{
			"prefix": "dnsbl",
			"ip":     ip,
		})

		var reversed [5]string
		reversed[4] = server
		for i, part := range strings.Split(ip.String(), ".") {
			reversed[3-i] = part
		}

		records, err := dns.QueryA(strings.Join(reversed[:], "."))

		if err != nil {
			log.Warn(err)
			return nil, err
		}

		if len(records) > 0 {
			log.Debug("found sender in the blacklist")

			return &Result{
				Reject: true,
				Code:   550,
				Text:   "I heard of you in the news!",
			}, nil
		}

		log.Debug("no match in the blacklist")
		return &Result{}, nil
	}
}
