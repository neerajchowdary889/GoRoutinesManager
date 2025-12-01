package types

import (
	"sync"
)

var (
	defaultRootOnce sync.Once
	defaultRoot     *Root
)

// DefaultRoot returns the default root instance.
// This provides a singleton-like convenience for simple applications,
// but is optional - you can always create your own Root with NewRoot().
//
// The default root is lazily initialized on first access.
//
// Example (convenience API):
//   app := types.DefaultRoot().App("payments")
//
// Example (explicit API - recommended for tests):
//   root := types.NewRoot()
//   app := root.App("payments")
func DefaultRoot() *Root {
	defaultRootOnce.Do(func() {
		defaultRoot = NewRoot()
	})
	return defaultRoot
}

// ResetDefaultRoot resets the default root (useful for testing).
// This is exported for testing purposes only.
// WARNING: Only call this in tests, never in production code.
func ResetDefaultRoot() {
	defaultRootOnce = sync.Once{}
	defaultRoot = nil
}

