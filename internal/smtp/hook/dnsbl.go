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
	"encoding/hex"
	"fmt"
	"net"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
	"github.com/lukasdietrich/briefmail/internal/mails"
)

func makeDnsblHook() FromHook {
	server := viper.GetString("hook.dnsbl.server")
	log.Info().
		Str("server", server).
		Msg("registering dnsbl hook")

	return func(ctx context.Context, submission bool, ip net.IP, _ mails.Address) (*Result, error) {
		if submission {
			log.InfoContext(ctx).
				Stringer("ip", ip).
				Msg("skipping dnsbl, because it is a submission")
			return &Result{}, nil
		}

		host := formatReverseIP(ip) + server
		log.InfoContext(ctx).
			Stringer("ip", ip).
			Str("host", host).
			Msg("looking up dnsbl")

		records, err := net.LookupIP(host)
		if err != nil {
			dnsErr, ok := err.(*net.DNSError)
			if !ok || !dnsErr.IsNotFound {
				return nil, fmt.Errorf("could not look up dnsbl: %w", err)
			}
		}

		if len(records) > 0 {
			log.InfoContext(ctx).
				Stringer("ip", ip).
				Msg("senders ip is blacklisted. rejecting request")

			return &Result{
				Reject: true,
				Code:   550,
				Text:   "I heard of you in the news!",
			}, nil
		}

		log.InfoContext(ctx).
			Stringer("ip", ip).
			Msg("snders ip is not blacklisted")

		return &Result{}, nil
	}
}

// formatReverseIP reverses an ip address to be used in a dnsbl lookup.
// The result ends in a trailing dot.
func formatReverseIP(ip net.IP) string {
	if ipv4 := ip.To4(); ipv4 != nil {
		// Reverse IPv4 octets (see RFC#5782 2.1.)

		const bufLen = len("255.255.255.255.")
		var (
			octs = make([]byte, bufLen)
			j    int
		)

		for i := 3; i >= 0; i-- {
			switch b := ipv4[i]; true {
			case b < 10:
				octs[j] = b + '0'
				j++

			case b < 100:
				octs[j] = b/10 + '0'
				octs[j+1] = b%10 + '0'
				j += 2

			default:
				octs[j] = b/100 + '0'
				octs[j+1] = (b/10)%10 + '0'
				octs[j+2] = b%10 + '0'
				j += 3
			}

			octs[j] = '.'
			j++
		}

		return string(octs[:j])
	}

	if ipv6 := ip.To16(); ipv6 != nil {
		// Reverse IPv6 nibbles (see RFC#5782 2.4.)

		const (
			hexLen = net.IPv6len * 2 // 1 byte = 2 hex letters
			bufLen = hexLen * 3      // original order + reverse order + dots
			offset = hexLen - 1      // offset for zero indexed reverse access
		)

		nibs := make([]byte, bufLen)
		hex.Encode(nibs, ipv6)

		for i := 0; i < hexLen; i++ {
			nibs[hexLen+i<<1] = nibs[offset-i]
			nibs[hexLen+i<<1+1] = '.'
		}

		return string(nibs[hexLen:])
	}

	return ""
}
