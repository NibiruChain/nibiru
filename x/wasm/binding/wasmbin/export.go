package wasmbin

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func GetPackageDir(t *testing.T) string {
	// Get the import path of the current package
	_, filename, _, _ := runtime.Caller(0)
	pkgDir := path.Dir(filename)
	pkgPath := path.Join(path.Base(pkgDir), "..")

	// Get the directory path of the package
	absPkgPath, err := filepath.Abs(pkgPath)
	require.NoError(t, err)
	return absPkgPath
}
