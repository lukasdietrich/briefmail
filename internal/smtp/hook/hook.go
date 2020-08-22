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

// HeaderField is a single mail header, that is to be set as a result of a hook.
type HeaderField struct {
	Key   string
	Value string
}

// Result is the result of calling a hook on incoming mail.
type Result struct {
	// Reject indicates if the mail should not be accepted for delivery.
	Reject bool
	// Headers is a list of headers to be prepended to incoming mail, if it is not rejected.
	Headers []HeaderField
	// Code is the smtp reply code used on rejection.
	Code int
	// Text is the smtp reply text used on rejection.
	Text string
}

// FromHook is a hook called during `MAIL`.
type FromHook func(context.Context, bool, net.IP, mails.Address) (*Result, error)

// DataHook is a hook called during `DATA`.
type DataHook func(context.Context, bool, io.Reader) (*Result, error)

// FromHooks creates all available and enabled FromHook implementations.
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

// DataHooks creates all available and enabled DataHook implementations.
func DataHooks() []DataHook {
	return nil
}
