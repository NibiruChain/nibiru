package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stopFunctions returns the node-stopping functions of the embedded localnet
// script, which sit between BEGIN and END markers.
func stopFunctions(t *testing.T) string {
	t.Helper()
	bz, err := os.ReadFile("localnet.sh")
	require.NoError(t, err)
	_, after, ok := strings.Cut(string(bz), "# BEGIN stop_running_nodes\n")
	require.True(t, ok)
	body, _, ok := strings.Cut(after, "# END stop_running_nodes\n")
	require.True(t, ok)
	return body
}

type fakeNode struct {
	cmd  *exec.Cmd
	done chan struct{}
}

func (f fakeNode) pid() string { return fmt.Sprint(f.cmd.Process.Pid) }

func (f fakeNode) exited() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// spawnFake runs bash under the name of the binary, so the process command
// line reads `<dir>/<name> -s -- <fake args>`.
func spawnFake(t *testing.T, dir, cwd, name string, fake ...string) fakeNode {
	t.Helper()
	bashPath, err := exec.LookPath("bash")
	require.NoError(t, err)
	bin := filepath.Join(dir, name)
	_ = os.Remove(bin)
	require.NoError(t, os.Symlink(bashPath, bin))
	// `bash -s` reads its script from stdin, so it waits on the open pipe and
	// the fake arguments become positional parameters.
	cmd := exec.Command(bin, append([]string{"-s", "--"}, fake...)...)
	cmd.Dir = cwd
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stdin.Close() })
	require.NoError(t, cmd.Start())
	f := fakeNode{cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(f.done) }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	return f
}

func runStopScript(t *testing.T, home, call string) (string, error) {
	t.Helper()
	script := "set -Eeuo pipefail\necho_error() { echo \"$@\" >&2; }\n" +
		"HOME=" + t.TempDir() + "\nCHAIN_DIR='" + home + "'\n" +
		stopFunctions(t) + "\n" + call + "\n"
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	return string(out), err
}

// The localnet script checks both binary names, matches the real start
// command, compares canonical homes, and waits for nodes to exit.
func TestLocalnetScriptStopsNodesUnderBothNames(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not found")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not found")
	}
	if _, err := os.Stat("/proc/self"); err != nil {
		t.Skip("no /proc")
	}
	dir := t.TempDir()
	root := t.TempDir()
	home := filepath.Join(root, "my home") // a space in the path
	require.NoError(t, os.MkdirAll(home, 0o755))
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(home, link))
	other := filepath.Join(root, "other")
	require.NoError(t, os.MkdirAll(other, 0o755))

	same := map[string]fakeNode{
		"nibiru start --home":          spawnFake(t, dir, root, "nibiru", "start", "--home", home),
		"nibid node start --home=/":    spawnFake(t, dir, root, "nibid", "node", "start", "--home="+home+"/"),
		"dot suffix":                   spawnFake(t, dir, root, "nibid", "start", "--home", home+"/."),
		"symlink":                      spawnFake(t, dir, root, "nibiru", "start", "--home="+link),
		"relative":                     spawnFake(t, dir, root, "nibiru", "start", "--home", "./my home/../my home"),
		"home before command":          spawnFake(t, dir, root, "nibid", "--home", home, "start"),
		"flags after start":            spawnFake(t, dir, root, "nibiru", "start", "--pruning", "nothing", "--home", home),
		"flags between node and start": spawnFake(t, dir, root, "nibiru", "node", "--home", home, "start"),
		"flags at every level":         spawnFake(t, dir, root, "nibid", "--log_level", "debug", "node", "--home="+home, "--trace", "start", "--pruning", "nothing"),
		"unknown flag before start":    spawnFake(t, dir, root, "nibid", "--future-flag", "x", "start", "--home", home),
	}
	untouched := map[string]fakeNode{
		"other home":                spawnFake(t, dir, root, "nibiru", "start", "--home", other),
		"query":                     spawnFake(t, dir, root, "nibid", "q", "bank", "--home", home),
		"start as an arg":           spawnFake(t, dir, root, "nibid", "keys", "add", "start", "--home", home),
		"keys add start, any flags": spawnFake(t, dir, root, "nibiru", "--home", home, "keys", "add", "start"),
		"start in a value":          spawnFake(t, dir, root, "nibid", "q", "wasm", "--node", "start", "--home", home),
	}
	time.Sleep(400 * time.Millisecond)

	out, err := runStopScript(t, home, "running_node_pids")
	require.NoError(t, err, out)
	want := []string{}
	for _, f := range same {
		want = append(want, f.pid())
	}
	for name, f := range same {
		require.Contains(t, strings.Fields(out), f.pid(), name)
	}
	require.ElementsMatch(t, want, strings.Fields(out))

	out, err = runStopScript(t, home, "stop_running_nodes")
	require.NoError(t, err, out)
	for name, f := range same {
		require.Eventually(t, f.exited, 5*time.Second, 50*time.Millisecond, name)
	}
	for name, f := range untouched {
		require.False(t, f.exited(), name)
	}
}

// The default home applies when a node has no --home flag, and a node whose
// home cannot be determined aborts the script.
func TestLocalnetScriptDefaultHomeAndUnsafe(t *testing.T) {
	if _, err := os.Stat("/proc/self"); err != nil {
		t.Skip("no /proc")
	}
	dir := t.TempDir()
	// runStopScript sets HOME to a fresh temp dir; read it back to match.
	script := func(call string) string {
		return "set -Eeuo pipefail\necho_error() { echo \"$@\" >&2; }\n" +
			"HOME='" + dir + "'\nCHAIN_DIR=\"$HOME/.nibid\"\n" +
			stopFunctions(t) + "\n" + call + "\n"
	}
	def := spawnFake(t, dir, dir, "nibiru", "start")
	time.Sleep(300 * time.Millisecond)
	out, err := exec.Command("bash", "-c", script("running_node_pids")).Output()
	require.NoError(t, err)
	require.Equal(t, []string{def.pid()}, strings.Fields(string(out)))

	// A --home flag without a value cannot be resolved: abort, kill nothing.
	_ = def.cmd.Process.Kill()
	require.Eventually(t, def.exited, 5*time.Second, 50*time.Millisecond)
	bad := spawnFake(t, dir, dir, "nibid", "start", "--home")
	time.Sleep(300 * time.Millisecond)
	co, err := exec.Command("bash", "-c", script("stop_running_nodes")).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(co), "could not be determined")
	require.False(t, bad.exited())
}

// A process that holds a file open under <home>/data and is not an identified
// node blocks the reset: the script names it, exits non-zero, and kills and
// deletes nothing. An identified node that holds the files is stopped as a
// node, and the reset proceeds.
func TestLocalnetScriptHoldersBlockButNodesStop(t *testing.T) {
	if _, err := os.Stat("/proc/self"); err != nil {
		t.Skip("no /proc")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data"), 0o755))
	lock := filepath.Join(home, "data", "LOCK")
	require.NoError(t, os.WriteFile(lock, nil, 0o644))
	elsewhere := filepath.Join(root, "other", "data", "LOCK")
	require.NoError(t, os.MkdirAll(filepath.Dir(elsewhere), 0o755))
	require.NoError(t, os.WriteFile(elsewhere, nil, 0o644))
	// A link to the home must resolve to the same data directory.
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(home, link))

	// A holder runs bash with the file open on fd 3 and no child that would
	// inherit it. It waits on a stdin pipe. name is the process name.
	holder := func(name, path string) fakeNode {
		bashPath, err := exec.LookPath("bash")
		require.NoError(t, err)
		bin := filepath.Join(t.TempDir(), name)
		require.NoError(t, os.Symlink(bashPath, bin))
		cmd := exec.Command(bin, "-c", `exec 3<"$1"; read -r _`, "x", path)
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		t.Cleanup(func() { _ = stdin.Close() })
		require.NoError(t, cmd.Start())
		f := fakeNode{cmd: cmd, done: make(chan struct{})}
		go func() { _ = cmd.Wait(); close(f.done) }()
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		return f
	}
	hold := holder("backup-tool", lock)
	away := holder("backup-tool", elsewhere)
	time.Sleep(400 * time.Millisecond)

	// The scan sees the holder, through the symlink too.
	for _, chain := range []string{home, link} {
		out, err := runStopScript(t, chain, "open_file_pids")
		require.NoError(t, err, out)
		require.Equal(t, []string{hold.pid()}, strings.Fields(out), chain)
	}

	// The holder is not a node: the script aborts, names it, and kills nothing.
	out, err := runStopScript(t, link, "stop_running_nodes; echo RESET")
	require.Error(t, err)
	require.Contains(t, out, hold.pid())
	require.Contains(t, out, "backup-tool")
	require.NotContains(t, out, "RESET")
	require.False(t, hold.exited(), "a plain holder must not be killed")
	require.False(t, away.exited())
	_, statErr := os.Stat(lock)
	require.NoError(t, statErr, "the data directory must stay")

	// Once the holder is gone, the reset proceeds.
	require.NoError(t, hold.cmd.Process.Kill())
	require.Eventually(t, hold.exited, 5*time.Second, 50*time.Millisecond)

	// An identified node that holds the files is stopped as a node, and the
	// script finishes. `bash -s` reads its script from stdin, so its command
	// line reads `nibiru -s -- start --home H`.
	bashPath, err := exec.LookPath("bash")
	require.NoError(t, err)
	bin := filepath.Join(t.TempDir(), "nibiru")
	require.NoError(t, os.Symlink(bashPath, bin))
	cmd := exec.Command(bin, "-s", "--", "start", "--home", home)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stdin.Close() })
	require.NoError(t, cmd.Start())
	node := fakeNode{cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(node.done) }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	_, err = stdin.Write([]byte("exec 3<'" + lock + "'\nread -r _\n"))
	require.NoError(t, err)
	time.Sleep(400 * time.Millisecond)

	out, err = runStopScript(t, home, "open_file_pids")
	require.NoError(t, err, out)
	require.Equal(t, []string{node.pid()}, strings.Fields(out))

	out, err = runStopScript(t, home, "stop_running_nodes; echo RESET")
	require.NoError(t, err, out)
	require.Contains(t, out, "RESET")
	require.True(t, node.exited())
	require.False(t, away.exited())
}
