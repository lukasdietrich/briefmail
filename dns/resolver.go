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

package dns

import "github.com/miekg/dns"

type Resolver struct {
	addr string
}

func NewResolver(addr string) *Resolver {
	return &Resolver{
		addr: addr,
	}
}

func (r *Resolver) query(domain string, t uint16) (*dns.Msg, error) {
	m := dns.Msg{
		Question: []dns.Question{{
			Name:   dns.Fqdn(domain),
			Qtype:  t,
			Qclass: dns.ClassINET,
		}},
	}

	return dns.Exchange(&m, r.addr)
}

func (r *Resolver) QueryA(domain string) ([]*dns.A, error) {
	res, err := r.query(domain, dns.TypeA)
	if err != nil {
		return nil, err
	}

	records := make([]*dns.A, 0, len(res.Answer))

	for _, rr := range res.Answer {
		if r, ok := rr.(*dns.A); ok {
			records = append(records, r)
		}
	}

	return records, err
}

func (r *Resolver) QueryMX(domain string) ([]*dns.MX, error) {
	res, err := r.query(domain, dns.TypeMX)
	if err != nil {
		return nil, err
	}

	records := make([]*dns.MX, 0, len(res.Answer))

	for _, rr := range res.Answer {
		if r, ok := rr.(*dns.MX); ok {
			records = append(records, r)
		}
	}

	return records, err
}
