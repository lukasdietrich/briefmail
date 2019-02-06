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

	"github.com/lukasdietrich/briefmail/dns"
	"github.com/lukasdietrich/briefmail/model"
)

func CheckDNSBL(server string) FromHook {
	return func(submission bool, ip net.IP, _ *model.Address) (*Result, error) {
		if submission || ip.To4() == nil {
			return &Result{}, nil
		}

		var reversed [5]string
		reversed[4] = server
		for i, part := range strings.Split(ip.String(), ".") {
			reversed[3-i] = part
		}

		records, err := dns.QueryA(strings.Join(reversed[:], "."))
		if err != nil {
			return nil, err
		}

		if len(records) > 0 {
			return &Result{
				Reject: true,
				Code:   550,
				Text:   "I heard of you in the news!",
			}, nil
		}

		return &Result{}, nil
	}
}
