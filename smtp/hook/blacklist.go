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

	"github.com/miekg/dns"

	"github.com/lukasdietrich/briefmail/model"
)

const (
	dnsAddress = "1.1.1.1:53" // Cloudflare public dns
)

func Blacklist(dnsbl string) FromHook {
	dnsbl = dns.Fqdn(dnsbl)

	return func(ip net.IP, _ *model.Address) (*Result, error) {
		if ip.To4() == nil {
			return &Result{}, nil
		}

		var reversed [5]string
		reversed[4] = dnsbl
		for i, part := range strings.Split(ip.String(), ".") {
			reversed[3-i] = part
		}

		var req dns.Msg
		req.SetQuestion(strings.Join(reversed[:], "."), dns.TypeA)

		res, err := dns.Exchange(&req, dnsAddress)
		if err != nil {
			return nil, err
		}

		for _, answer := range res.Answer {
			switch answer.(type) {
			case *dns.A:
				return &Result{
					Reject: true,
					Code:   550,
					Text:   "I heard of you in the news!",
				}, nil
			}
		}

		return &Result{}, nil
	}
}
