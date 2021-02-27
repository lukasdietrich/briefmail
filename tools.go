// +build tools

package tools

import (
	// Linter
	_ "golang.org/x/lint/golint"

	// Dependency injection
	_ "github.com/google/wire/cmd/wire"

	// Mock generation
	_ "github.com/vektra/mockery/v2"
)
