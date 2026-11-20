package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathProbe holds the file-system operations ReportPaths needs. OSPathProbe
// returns the real ones.
type PathProbe struct {
	// IsExecutable reports whether path is a regular file, or a link to one,
	// with an execute bit set.
	IsExecutable func(path string) bool
	// Resolve returns path with every symbolic link followed.
	Resolve func(path string) (string, error)
}

// OSPathProbe probes the real file system.
func OSPathProbe() PathProbe {
	return PathProbe{
		IsExecutable: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.Mode().IsRegular() &&
				info.Mode().Perm()&0o111 != 0
		},
		Resolve: filepath.EvalSymlinks,
	}
}

// PathMatch is one executable found on PATH.
type PathMatch struct {
	// Path is the directory entry on PATH, such as ~/.bun/bin/nibiru.
	Path string `json:"path" yaml:"path"`
	// Resolved is Path with symbolic links followed.
	Resolved string `json:"resolved" yaml:"resolved"`
	// IsRunning is true when Resolved is the running binary. A nibid symlink to
	// the running nibiru file counts as the running binary, not as a conflict.
	IsRunning bool `json:"is_running" yaml:"is_running"`
}

// NamePaths lists the PATH matches for one command name in PATH order. The
// first match is the file the shell runs when you type the name.
type NamePaths struct {
	Name    string      `json:"name" yaml:"name"`
	Matches []PathMatch `json:"matches" yaml:"matches"`
	// FirstIsRunning is true when the first match is the running binary. It is
	// also true when the name is not on PATH at all, because nothing shadows
	// the running binary.
	FirstIsRunning bool `json:"first_is_running" yaml:"first_is_running"`
}

// PathReport says where the running binary lives and which executables with
// the same command names sit on PATH.
type PathReport struct {
	// RunningFrom is the running binary with symbolic links followed.
	RunningFrom string      `json:"running_from" yaml:"running_from"`
	Names       []NamePaths `json:"commands" yaml:"commands"`
}

// ReportPaths finds the executables called names in the directories of
// pathEnv, in order, and marks the ones that resolve to running. Empty PATH
// entries and non-executable files are skipped.
func ReportPaths(
	running, pathEnv string, names []string, probe PathProbe,
) PathReport {
	resolve := func(p string) string {
		if r, err := probe.Resolve(p); err == nil {
			return r
		}
		return p
	}
	report := PathReport{RunningFrom: resolve(running)}

	for _, name := range names {
		np := NamePaths{Name: name, FirstIsRunning: true}
		seen := map[string]bool{}
		for _, dir := range filepath.SplitList(pathEnv) {
			if dir == "" {
				continue
			}
			candidate := filepath.Join(dir, name)
			if seen[candidate] || !probe.IsExecutable(candidate) {
				continue
			}
			seen[candidate] = true
			resolved := resolve(candidate)
			np.Matches = append(np.Matches, PathMatch{
				Path:      candidate,
				Resolved:  resolved,
				IsRunning: resolved == report.RunningFrom,
			})
		}
		if len(np.Matches) > 0 {
			np.FirstIsRunning = np.Matches[0].IsRunning
		}
		report.Names = append(report.Names, np)
	}
	return report
}

// AllFirstRunning is true when no name is shadowed by another file on PATH.
func (r PathReport) AllFirstRunning() bool {
	for _, np := range r.Names {
		if !np.FirstIsRunning {
			return false
		}
	}
	return true
}

// Lines renders the report as plain text. Matches that are the running binary
// are omitted, so a clean install prints one line.
func (r PathReport) Lines() []string {
	lines := []string{"running from: " + r.RunningFrom}
	for _, np := range r.Names {
		for i, m := range np.Matches {
			if m.IsRunning {
				continue
			}
			line := "also on PATH: " + m.Path
			if i == 0 {
				line += fmt.Sprintf("  (runs when you type %q)", np.Name)
			}
			lines = append(lines, line)
		}
	}
	return lines
}

// String joins Lines with newlines.
func (r PathReport) String() string { return strings.Join(r.Lines(), "\n") }
