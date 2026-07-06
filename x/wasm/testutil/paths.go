package testutil

import (
	"path/filepath"
	"runtime"
)

// RepoPath returns an absolute path under the repository root.
func RepoPath(elem ...string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to locate x/wasm/testutil path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	parts := append([]string{root}, elem...)
	return filepath.Join(parts...)
}

// FixturePath returns an absolute path under x/wasm.
func FixturePath(elem ...string) string {
	parts := append([]string{"x", "wasm"}, elem...)
	return RepoPath(parts...)
}
