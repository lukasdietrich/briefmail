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

package pop3

import "sync"

type locks struct {
	entries map[int64]bool
	mu      sync.Mutex
}

func newLocks() *locks {
	return &locks{
		entries: make(map[int64]bool),
	}
}

func (l *locks) lock(key int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.entries[key] {
		return false
	}

	l.entries[key] = true
	return true
}

func (l *locks) unlock(key int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key)
}
