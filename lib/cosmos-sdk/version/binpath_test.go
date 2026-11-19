package version_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"
)

// writeBin creates an executable (or not) file in dir and returns its path.
func writeBin(t *testing.T, dir, name string, exec bool) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	mode := os.FileMode(0o644)
	if exec {
		mode = 0o755
	}
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), mode))
	return path
}

func pathOf(dirs ...string) string { return joinList(dirs) }

func joinList(dirs []string) string {
	out := ""
	for i, d := range dirs {
		if i > 0 {
			out += string(os.PathListSeparator)
		}
		out += d
	}
	return out
}

func find(t *testing.T, r version.PathReport, name string) version.NamePaths {
	t.Helper()
	for _, np := range r.Names {
		if np.Name == name {
			return np
		}
	}
	t.Fatalf("no report for %s", name)
	return version.NamePaths{}
}

func TestReportPaths(t *testing.T) {
	names := []string{"nibiru", "nibid"}
	probe := version.OSPathProbe()

	t.Run("no other copy", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		running := writeBin(t, goBin, "nibiru", true)
		require.NoError(t, os.Symlink("nibiru", filepath.Join(goBin, "nibid")))

		r := version.ReportPaths(running, pathOf(goBin), names, probe)
		require.Equal(t, running, r.RunningFrom)
		for _, name := range names {
			np := find(t, r, name)
			require.True(t, np.FirstIsRunning, name)
			require.Len(t, np.Matches, 1, name)
			require.True(t, np.Matches[0].IsRunning, name)
		}
		// The nibid symlink to the same file is not a conflict.
		require.Equal(t, []string{"running from: " + running}, r.Lines())
	})

	t.Run("shim earlier on PATH", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		bunBin := filepath.Join(root, "bun")
		running := writeBin(t, goBin, "nibiru", true)
		shim := writeBin(t, bunBin, "nibiru", true)

		r := version.ReportPaths(running, pathOf(bunBin, goBin), names, probe)
		np := find(t, r, "nibiru")
		require.False(t, np.FirstIsRunning)
		require.Equal(t, shim, np.Matches[0].Path)
		require.False(t, np.Matches[0].IsRunning)
		require.True(t, np.Matches[1].IsRunning)
		require.Equal(t, []string{
			"running from: " + running,
			"also on PATH: " + shim + `  (runs when you type "nibiru")`,
		}, r.Lines())
	})

	t.Run("shim later on PATH", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		bunBin := filepath.Join(root, "bun")
		running := writeBin(t, goBin, "nibiru", true)
		shim := writeBin(t, bunBin, "nibiru", true)

		r := version.ReportPaths(running, pathOf(goBin, bunBin), names, probe)
		np := find(t, r, "nibiru")
		require.True(t, np.FirstIsRunning)
		require.Equal(t, []string{
			"running from: " + running,
			"also on PATH: " + shim,
		}, r.Lines())
	})

	t.Run("shim symlink resolving to the running binary", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		other := filepath.Join(root, "other")
		running := writeBin(t, goBin, "nibiru", true)
		require.NoError(t, os.MkdirAll(other, 0o755))
		require.NoError(t, os.Symlink(running, filepath.Join(other, "nibiru")))

		r := version.ReportPaths(running, pathOf(other, goBin), names, probe)
		np := find(t, r, "nibiru")
		require.True(t, np.FirstIsRunning)
		require.Len(t, r.Lines(), 1)
	})

	t.Run("running through the nibid symlink", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		real := writeBin(t, goBin, "nibiru", true)
		link := filepath.Join(goBin, "nibid")
		require.NoError(t, os.Symlink("nibiru", link))

		r := version.ReportPaths(link, pathOf(goBin), names, probe)
		resolved, err := filepath.EvalSymlinks(real)
		require.NoError(t, err)
		require.Equal(t, resolved, r.RunningFrom)
		require.Len(t, r.Lines(), 1)
	})

	t.Run("empty PATH", func(t *testing.T) {
		running := writeBin(t, t.TempDir(), "nibiru", true)
		r := version.ReportPaths(running, "", names, probe)
		require.Equal(t, running, r.RunningFrom)
		for _, np := range r.Names {
			require.Empty(t, np.Matches)
			require.True(t, np.FirstIsRunning)
		}
		require.Len(t, r.Lines(), 1)
	})

	t.Run("non-executable file with the same name is ignored", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		docs := filepath.Join(root, "docs")
		running := writeBin(t, goBin, "nibiru", true)
		writeBin(t, docs, "nibiru", false)

		r := version.ReportPaths(running, pathOf(docs, goBin), names, probe)
		np := find(t, r, "nibiru")
		require.Len(t, np.Matches, 1)
		require.True(t, np.FirstIsRunning)
	})

	t.Run("duplicate PATH entries list once", func(t *testing.T) {
		root := t.TempDir()
		goBin := filepath.Join(root, "gobin")
		running := writeBin(t, goBin, "nibiru", true)
		r := version.ReportPaths(running, pathOf(goBin, goBin, ""), names, probe)
		require.Len(t, find(t, r, "nibiru").Matches, 1)
	})
}
