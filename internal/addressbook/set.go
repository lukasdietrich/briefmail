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

package addressbook

type Set struct {
	entries map[string]bool
}

func NewSet(s []string, mapping func(string) (string, error)) (*Set, error) {
	entries := make(map[string]bool)

	for _, e := range s {
		n, err := mapping(e)
		if err != nil {
			return nil, err
		}

		entries[n] = true
	}

	return &Set{entries}, nil
}

func (s *Set) Contains(e string) bool {
	return s.entries[e]
}
