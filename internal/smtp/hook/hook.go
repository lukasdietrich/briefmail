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
	"io"
	"net"

	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/mails"
)

func init() {
	viper.SetDefault("hook.spf.enable", true)

	viper.SetDefault("hook.dnsbl.enable", false)
	viper.SetDefault("hook.dnsbl.server", "zen.spamhaus.org")
}

type HeaderField struct {
	Key   string
	Value string
}

type Result struct {
	Reject bool

	Headers []HeaderField
	Code    int
	Text    string
}

type FromHook func(bool, net.IP, mails.Address) (*Result, error)
type DataHook func(bool, io.Reader) (*Result, error)

func FromHooks() []FromHook {
	var hooks []FromHook

	for key, makeHook := range map[string](func() FromHook){
		"spf":   makeSpfHook,
		"dnsbl": makeDnsblHook,
	} {
		if viper.GetBool("hook." + key + ".enable") {
			hooks = append(hooks, makeHook())
		}
	}

	return hooks
}

func DataHooks() []DataHook {
	return nil
}
