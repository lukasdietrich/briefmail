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
	"context"
	"fmt"
	"net"

	"github.com/zaccone/spf"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/models"
)

func makeSpfHook() FromHook {
	log.Info().Msg("registering spf hook")

	return func(ctx context.Context, submission bool, ip net.IP, from models.Address) (*Result, error) {
		if submission {
			return &Result{}, nil
		}

		log.InfoContext(ctx).
			Stringer("from", from).
			Msg("looking up spf")

		result, _, err := spf.CheckHost(ip, from.Domain(), from.String())
		if err != nil {
			log.InfoContext(ctx).
				Stringer("from", from).
				Err(err).
				Msg("could not check spf")
		} else {
			log.InfoContext(ctx).
				Stringer("from", from).
				Stringer("result", result).
				Msg("spf result")
		}

		if result == spf.Fail {
			return &Result{
				Reject: true,
				Code:   550,
				Text:   "you shall not pass",
			}, nil
		}

		return &Result{
			Reject: false,
			Headers: []HeaderField{
				{
					Key: "Received-SPF",
					Value: fmt.Sprintf(
						"%s (with domain=%s of sender=%s) client-ip=%s;",
						result, from.Domain(), from.String(), ip),
				},
			},
		}, nil
	}
}
