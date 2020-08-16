// +build tools

package tools

import (
	// Linter
	_ "golang.org/x/lint/golint"

	// Dependency injection
	_ "github.com/google/wire/cmd/wire"

	// Database migrations
	_ "github.com/rubenv/sql-migrate/sql-migrate"
)
