// Package rux's tests live under internal/v2/.
// This file exists to keep the package importable for `go test ./...`.
package rux

import "testing"

func TestPackageImportable(t *testing.T) {
	// Compile-only test — the real tests are in internal/v2.
}
