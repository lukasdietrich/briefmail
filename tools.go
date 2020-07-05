// +build tools

package tools

import (
	// Linter
	_ "golang.org/x/lint/golint"

	// Dependency Injection
	_ "github.com/google/wire/cmd/wire"
)
