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

package config

import "github.com/lukasdietrich/briefmail/smtp/hook"

type Hook struct {
	SPF struct {
		Enable bool
	}

	DNSBL struct {
		Enable bool
		Server string
	}
}

func (h *Hook) MakeInstances() ([]hook.FromHook, []hook.DataHook, error) {
	var (
		from []hook.FromHook
		data []hook.DataHook
	)

	if c := h.SPF; c.Enable {
		from = append(from, hook.CheckSPF())
	}

	if c := h.DNSBL; c.Enable {
		from = append(from, hook.CheckDNSBL(c.Server))
	}

	return from, data, nil
}
