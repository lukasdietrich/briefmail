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
