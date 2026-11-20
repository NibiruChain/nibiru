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

// The localnet script checks both binary names, only for nodes that use the
// target home, and waits for them to exit before it resets.
func TestLocalnetScriptStopsNodesUnderBothNames(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}
	if _, err := exec.LookPath("ps"); err != nil {
		t.Skip("ps not found")
	}
	dir := t.TempDir()

	// A fake node is bash run under the name of the binary, so ps shows
	// `<dir>/nibiru -c ... x start --home H`. The trailing `:` keeps bash
	// from replacing itself with sleep.
	spawn := func(name string, fake ...string) fakeNode {
		bin := filepath.Join(dir, name)
		_ = os.Remove(bin)
		require.NoError(t, os.Symlink(bashPath, bin))
		cmd := exec.Command(bin,
			append([]string{"-c", "sleep 60; :", "x"}, fake...)...)
		require.NoError(t, cmd.Start())
		f := fakeNode{cmd: cmd, done: make(chan struct{})}
		go func() { _ = cmd.Wait(); close(f.done) }()
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		return f
	}
	home, other := t.TempDir(), t.TempDir()
	nibiru := spawn("nibiru", "start", "--home", home)
	nibid := spawn("nibid", "node", "start", "--home="+home+"/")
	elsewhere := spawn("nibiru", "start", "--home", other)
	query := spawn("nibid", "q", "bank", "--home", home)
	time.Sleep(300 * time.Millisecond)

	prelude := "set -Eeuo pipefail\necho_error() { echo \"$@\" >&2; }\n" +
		"CHAIN_DIR=" + home + "\n" + stopFunctions(t)

	out, err := exec.Command("bash", "-c", prelude+"running_node_pids\n").Output()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{nibiru.pid(), nibid.pid()},
		strings.Fields(string(out)))

	o, err := exec.Command("bash", "-c", prelude+"stop_running_nodes\n").CombinedOutput()
	require.NoError(t, err, string(o))
	require.Eventually(t, func() bool { return nibiru.exited() && nibid.exited() },
		5*time.Second, 50*time.Millisecond)
	require.False(t, elsewhere.exited(), "node on another home must keep running")
	require.False(t, query.exited(), "a query must keep running")
}
