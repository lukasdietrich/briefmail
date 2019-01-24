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
	"fmt"
	"net"

	"github.com/zaccone/spf"

	"github.com/lukasdietrich/briefmail/model"
)

func CheckSPF() FromHook {
	return func(submission bool, ip net.IP, from *model.Address) (*Result, error) {
		if submission {
			return &Result{}, nil
		}

		result, _, err := spf.CheckHost(ip, from.Domain, from.String())

		switch result {
		case spf.Temperror, spf.Permerror:
			return nil, err

		case spf.Fail:
			return &Result{
				Reject: true,
				Code:   550,
				Text:   "you shall not pass",
			}, nil

		default:
			return &Result{
				Reject: false,
				Headers: []HeaderField{
					{
						Key: "Received-SPF",
						Value: fmt.Sprintf(
							"%s (with domain=%s of sender=%s) client-ip=%s;",
							result, from.Domain, from.String(), ip),
					},
				},
			}, nil
		}
	}
}
