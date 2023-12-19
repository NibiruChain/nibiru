package testutil

import (
	"path"
	"path/filepath"
	"runtime"
)

// GetPackageDir: Returns the absolute path of the Golang package that
// calls this function.
func GetPackageDir() (string, error) {
	// Get the import path of the current package
	_, filename, _, _ := runtime.Caller(0)
	pkgDir := path.Dir(filename)
	pkgPath := path.Join(path.Base(pkgDir), "..")

	// Get the directory path of the package
	return filepath.Abs(pkgPath)
}
