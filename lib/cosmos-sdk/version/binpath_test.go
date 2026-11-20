package version_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

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

// The long text output is one YAML document, even with several other copies
// on PATH and a path that holds ": ".
func TestRenderLongParsesAsYAMLAndJSON(t *testing.T) {
	root := t.TempDir()
	running := writeBin(t, filepath.Join(root, "gobin"), "nibiru", true)
	dirs := []string{
		filepath.Join(root, "bun: bin"),
		filepath.Join(root, "old"),
		filepath.Join(root, "usr"),
		filepath.Join(root, "gobin"),
	}
	for _, d := range dirs[:3] {
		writeBin(t, d, "nibiru", true)
		writeBin(t, d, "nibid", true)
	}
	report := version.ReportPaths(running, pathOf(dirs...),
		[]string{"nibiru", "nibid"}, version.OSPathProbe())
	require.False(t, report.AllFirstRunning())

	info := version.NewInfo()
	for _, output := range []string{"text", "json"} {
		bz, err := version.RenderLong(info, &report, output)
		require.NoError(t, err)

		var got map[string]any
		if output == "json" {
			require.NoError(t, json.Unmarshal(bz, &got))
		} else {
			// Strict mode rejects duplicate keys.
			require.NoError(t, yaml.UnmarshalStrict(bz, &got), string(bz))
			require.Equal(t, 1, strings.Count(string(bz), "binary_paths:"))
			require.NotContains(t, string(bz), "also on PATH")
		}
		bp, ok := got["binary_paths"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, report.RunningFrom, bp["running_from"])
		require.Len(t, bp["commands"], 2)
		require.Equal(t, info.Version, got["version"])
	}

	// Existing fields keep their order in JSON, with binary_paths last.
	bz, err := version.RenderLong(info, &report, "json")
	require.NoError(t, err)
	require.Greater(t, strings.Index(string(bz), `"binary_paths"`),
		strings.Index(string(bz), `"version"`))
	require.True(t, strings.HasSuffix(strings.TrimSuffix(string(bz), "}"), "}"))
}
